// Copyright (c) The Arribada initiative.
// Licensed under the MIT License.

package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

//	"github.com/arribada/insight-360-common/pkg/common"
	"github.com/arribada/insight-360/common/pkg/datatype"
	"github.com/arribada/insight-360/common/pkg/utils"
	"github.com/arribada/insight-360/whisper/pkg/gc"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

var (
	flagSource        string
	flagOutDir        string
	flagModelpath     string
	flagStdout        bool
	flagNoBracket     bool
	flagHost          string
	flagVerbose       bool
	flagGCGracePeriod int // in seconds
	flagGCInterval    int // in seconds
	flagMountPoint    string
	flagDriveLabel    string
)

func init() {
	flag.StringVar(&flagOutDir, "outdir", "./out", "output data directory")
	flag.StringVar(&flagModelpath, "modelpath", "./data/models/ggml-base.en.bin", "path to the model file used for this run")
	flag.BoolVar(&flagStdout, "stdout", false, "write transcribed text to stdout in addition to the json file")
	flag.BoolVar(&flagVerbose, "v", false, "be verbose")
	flag.StringVar(&flagSource, "source", "http://localhost:8080", "Where to actively fetch audio from.")
	flag.BoolVar(&flagNoBracket, "nobracket", true, "Automatically filters out all non actual speech (music, silences, etc).")
	flag.StringVar(&flagHost, "host", ":8080", "host:port on which we receive start/stop messages")
	flag.IntVar(&flagGCGracePeriod, "gcgrace", 86400, "the period (in seconds) during which an audio file, after it's been created, is excluded from garbage collection")
	flag.IntVar(&flagGCInterval, "gcinterval", 3600, "the interval in between subsequent garbage collection runs")
	flag.StringVar(&flagMountPoint, "mnt", "", "the mount point for the external drive, if any. If specified, it overrides outdir.")
	flag.StringVar(&flagDriveLabel, "label", "", "the label of the external drive to be mounted.")
}

const (
	bitDepth         = 16
	initialBusyPause = 100 * time.Millisecond
	modelsRepo       = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main"
)

var (
	audioSource string

	pausedMu sync.RWMutex
	// whether to actually do work (fetching audio, and transcribing it), controlled
	// by start/stop signals from the network.
	paused bool = true

	resumeCh = make(chan struct{})

	model whisper.Model
	// FIFO stack of work to do for the transcriber. asynchronous.
	samplesChan chan smpls = make(chan smpls, 100)
	// sentinel error.
	noSamplesErr = errors.New("no samples to transcribe")
)

func main() {
	flag.Parse()

	if err := checkFlags(); err != nil {
		log.Fatal(err)
	}

	if err := initOutDir(); err != nil {
		log.Fatal(err)
	}

	var err error
//	audioSource, err = common.MResolve(flagSource)
	audioSource = flagSource
	if err != nil {
		log.Fatal(err)
	}

	// Starting start/stop server
	if flagHost != "" {
		go func() {
			http.Handle("/", &server{})
			log.Fatal(http.ListenAndServe(flagHost, nil))
		}()
	}

	go func() {
		if err := audioGC(); err != nil {
			log.Fatalf("Garbage Collector for audio failed: %v", err)
		}
	}()

	// Load the model
	model, err = whisper.New(flagModelpath)
	if err != nil {
		log.Fatalf("while loading whisper model: %v", err)
	}
	defer model.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Useless for now.
	defer cancel()
	go runTranscriber(ctx)

	factor := math.Pow(2, float64(bitDepth)-1)
	busyPause := initialBusyPause
	for {
		pausedMu.RLock()
		isPaused := paused
		pausedMu.RUnlock()
		if isPaused {
			// block here until we get restarted by request/button press
			<-resumeCh
		}

		asFloats, asInts, err := getSamples(factor)
		if err != nil {
			log.Printf("while getting audio samples: %v", err)
			continue
		}

		pausedMu.RLock()
		isPaused = paused
		pausedMu.RUnlock()
		if isPaused {
			// no need to start transcribing if we got paused while fetching audio
			continue
		}

		select {
		case samplesChan <- smpls{asFloats: asFloats, asInts: asInts}:
			busyPause = initialBusyPause
		default:
			if flagVerbose {
				log.Printf("dropping audio, transcriber too busy")
			}
			time.Sleep(busyPause)
			busyPause *= 2
		}
	}
}

type server struct{}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/start" && r.URL.Path != "/stop" {
		http.Error(w, "invalid command", 404)
		return
	}

	if flagVerbose {
		log.Printf("Got button request: %v", r.URL.Path)
	}

	defer func() {
		_, _ = w.Write([]byte("OK"))
	}()

	if r.URL.Path == "/start" {
		pausedMu.RLock()
		isPaused := paused
		pausedMu.RUnlock()
		if !isPaused {
			// already running, nothing to do
			return
		}

		select {
		case resumeCh <- struct{}{}:
			pausedMu.Lock()
			paused = false
			pausedMu.Unlock()
		case <-time.After(time.Second):
			log.Printf("timed out on start signal")
		}
		return
	}

	if r.URL.Path == "/stop" {
		pausedMu.Lock()
		paused = true
		pausedMu.Unlock()
		return
	}
}

func initOutDir() error {
	if flagMountPoint == "" {
		return os.MkdirAll(flagOutDir, 0750)
	}

	if err := utils.MountSSD(flagDriveLabel, flagMountPoint); err != nil {
		if err != utils.ErrAlreadyMounted {
			return fmt.Errorf("while trying to mount %v: %v", flagDriveLabel, err)
		}
		if flagVerbose {
			log.Printf("%v already mounted", flagDriveLabel)
		}
	}

	return nil
}

func checkFlags() error {
	if flagMountPoint != "" && flagDriveLabel == "" || flagMountPoint == "" && flagDriveLabel != "" {
		return fmt.Errorf("Both (or neither) of -mount and -label need to be specified")
	}

	if flagMountPoint != "" {
		if flagVerbose {
			log.Printf("-outdir overridden by -mount")
		}
		flagOutDir = flagMountPoint
	}

	if _, err := os.Stat(flagModelpath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("while checking -modelpath: %v", err)
		}
		if err := downloadModel(); err != nil {
			return fmt.Errorf("while downloading model: %v", err)
		}
	}

	if flagHost == "" {
		log.Printf("Running with no server for start/stop messages, because no -host specified")
	}

	return nil
}

func downloadModel() error {
	// TODO: let repo where to fetch them be configurable? or just hardcode our own? etc.
	destDir, modelName := path.Split(flagModelpath)
	if flagVerbose {
		log.Printf("Starting to download model: %v", modelName)
	}
	modelURL := modelsRepo + "/" + modelName
	resp, err := http.Get(modelURL)
	if err != nil {
		return fmt.Errorf("for model %v: %v", modelURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("for model %v, unexpected code: %v", modelURL, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("while getting model %v: %v", modelURL, err)
	}

	if err := os.MkdirAll(destDir, 0750); err != nil {
		return fmt.Errorf("while making dir for model: %v", err)
	}

	if err := os.WriteFile(flagModelpath, data, 0640); err != nil {
		return fmt.Errorf("while writing model file %v: %v", flagModelpath, err)
	}
	return nil
}

func getSamples(factor float64) ([]float32, []int, error) {
	var asFloats []float32
	var asInts []int

	// TODO: For now we trust the audio server that one request/response is the
	// amount of the data we want to transcribe. But that might change later.
	data, err := getAudio() // blocking call.
	if err != nil {
		return nil, nil, fmt.Errorf("while fetching audio: %w", err)
	}
	if len(data) == 0 {
		log.Fatal("TODO: investigate, no data from source")
	}

	rd := bytes.NewReader(data)
	for {
		i, err := binary.ReadVarint(rd)
		if err != nil {
			// TODO: revisit. Are both needed?
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				log.Fatalf("TODO: investigate. while decoding binary encoded audio data: %v", err)
			}
		}
		asInts = append(asInts, int(i))
		asFloats = append(asFloats, float32(float64(i)/factor))
		// End of current chunk of data
		if err != nil {
			break
		}
	}

	return asFloats, asInts, nil
}

func getAudio() ([]byte, error) {
	// this is unrelated to Retry-After, this is for dealing when the audio source
	// server is not up (yet), with an exponential back-off.
	retryPause := time.Second
	// no need to increase into crazy long durations
	maxPause := 5 * time.Minute

	// We keep on trying until streamer has a new chunk for us.
	// We trust Retry-After as a clue for when to retry.
	var res *http.Response
	var err error
	for {
		res, err = http.Get(audioSource)
		if err != nil {
			// TODO: assume for now that the only kind of error we get here is when the
			// server is not up
			if flagVerbose {
				log.Printf("audio at %v not available, retrying in %v seconds", audioSource, retryPause.Seconds())
			}
			time.Sleep(retryPause)
			retryPause *= 2
			if retryPause > maxPause {
				retryPause = maxPause
			}
			pausedMu.RLock()
			isPaused := paused
			pausedMu.RUnlock()
			if isPaused {
				return nil, errors.New("we got paused while retrying for audio")
			}
			continue
		}
		retryPause = time.Second
		code := res.StatusCode
		if code == 200 {
			break
		}
		res.Body.Close()
		if code != 429 {
			return nil, fmt.Errorf("unexpected code: %v", code)
		}
		retry := res.Header.Get("Retry-After")
		if retry == "" {
			return nil, fmt.Errorf("missing Retry-After header in audio response")
		}
		sleep, err := strconv.Atoi(retry)
		if err != nil {
			return nil, fmt.Errorf("while converting Retry-After header: %v", err)
		}
		time.Sleep(time.Duration(sleep) * time.Second)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("while reading audio data: %v", err)
	}
	// TODO: make sure we're efficient regarding TCP connection reuse.
	return data, nil
}

type smpls struct {
	asInts   []int
	asFloats []float32
}

type Segment struct {
	Start time.Duration
	End   time.Duration
	Text  string
}

func runTranscriber(ctx context.Context) {
	for {
		var samples smpls
		select {
		case <-ctx.Done():
			log.Printf("Transcriber terminating.")
			return
		case samples = <-samplesChan:
		}
		if flagVerbose {
			log.Printf("Starting a transcription. Work still in queue: %d", len(samplesChan))
		}

		if len(samples.asFloats) == 0 {
			if flagVerbose {
				log.Printf("skipping empty sample")
			}
			continue
		}

		tm := time.Now()

		go func() {
			if err := writeWav(tm, samples.asInts); err != nil {
				// TODO: consider fatal errors?
				log.Printf("wave writing error: %v", err)
			}
		}()

		segments, err := transcribe(samples.asFloats)
		if err != nil {
			if err == noSamplesErr {
				if flagVerbose {
					log.Printf("skipping empty sample")
				}
				continue
			}
			log.Printf("transcription error: %v", err)
			continue
		}
		// because it could be all blanks
		if len(segments) < 1 {
			if flagVerbose {
				log.Printf("skipping all blank segments")
			}
			continue
		}

		if flagStdout {
			writeSegments(segments)
		}

		if err := writeSTTFile(tm, segments); err != nil {
			// TODO: consider fatal errors?
			log.Printf("error while writing STT file: %v", err)
			continue
		}
	}
}

func transcribe(samples []float32) ([]Segment, error) {
	if len(samples) == 0 {
		return nil, noSamplesErr
	}

	// Process samples.
	context, err := model.NewContext()
	if err != nil {
		return nil, err
	}
	// TODO: keep the "last words", and prepend them to next iteration, to help with
	// "word boundary" issues. But maybe we even do that earlier.
	if err := context.Process(samples, nil, nil); err != nil {
		return nil, err
	}

	var segments []Segment
	for {
		segment, err := context.NextSegment()
		if err != nil {
			break
		}
		if flagNoBracket {
			if strings.HasPrefix(segment.Text, "(") && strings.HasSuffix(segment.Text, ")") ||
				strings.HasPrefix(segment.Text, "[") && strings.HasSuffix(segment.Text, "]") {
				if flagVerbose {
					log.Printf("skipping irrelevant segment: %v", segment.Text)
				}
				continue
			}
		}
		segments = append(segments, Segment{
			Start: segment.Start,
			End:   segment.End,
			Text:  segment.Text,
		})
	}
	return segments, nil
}

func writeSegments(segments []Segment) {
	for _, v := range segments {
		fmt.Printf("[%6s->%6s] %s\n", v.Start, v.End, v.Text)
	}
}

func writeSTTFile(timeStamp time.Time, segments []Segment) error {
	// write in a temp dir, because we want to finish with a rename in the fsnotify
	// watched dir.
	tmpDir := filepath.Join(flagOutDir, datatype.TmpDirName)
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return err
	}

	name, err := datatype.NewFileName(datatype.Speech, timeStamp)
	if err != nil {
		return err
	}
	name = filepath.Join(tmpDir, name)

	f, err := os.Create(name)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	if err := enc.Encode(segments); err != nil {
		f.Close()
		return fmt.Errorf("while encoding Segments to JSON: %v", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("while closing file %v: %v", name, err)
	}
	log.Printf("Successfully wrote STT file: %v", name)

	newName := filepath.Join(flagOutDir, filepath.Base(name))
	return os.Rename(name, newName)
}

func writeWav(timeStamp time.Time, asInts []int) error {
	// write in a temp dir, because we want to finish with a rename in the fsnotify
	// watched dir.
	tmpDir := filepath.Join(flagOutDir, datatype.TmpDirName)
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return err
	}

	name, err := datatype.NewFileName(datatype.Audio, timeStamp)
	if err != nil {
		return err
	}
	name = filepath.Join(tmpDir, name)

	buf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  16000,
		},
		Data:           asInts,
		SourceBitDepth: 16,
	}

	out, err := os.Create(name)
	if err != nil {
		return err
	}

	e := wav.NewEncoder(out,
		buf.Format.SampleRate,
		int(buf.SourceBitDepth),
		buf.Format.NumChannels,
		1)
	if err := e.Write(buf); err != nil {
		return err
	}
	if err = e.Close(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	newName := filepath.Join(flagOutDir, filepath.Base(name))
	return os.Rename(name, newName)
}

func audioGC() error {
	g := gc.GC{
		RootDir:     flagOutDir,
		Verbose:     flagVerbose,
		GracePeriod: flagGCGracePeriod,
	}
	for {
		log.Printf("Starting a GC run")
		if err := g.Run(); err != nil {
			return err
		}
		// TODO: do we need to make sure that a previous GC run is over before we start
		// a new one? In theory yes, if the interval is short enough. In practice probably no
		// need to worry about it.
		time.Sleep(time.Duration(flagGCInterval) * time.Second)
	}
}
