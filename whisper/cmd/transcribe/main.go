// Copyright (c) The Arribada initiative.
// Licensed under the MIT License.

package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/arribada/insight-360-common/pkg/common"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

var (
	flagHost    string
	flagVerbose bool
)

func init() {
	flag.BoolVar(&flagVerbose, "v", false, "be verbose")
	flag.StringVar(&flagHost, "host", ":8080", "host:port on which we receive start/stop messages")
}

func main() {
	flag.Parse()

	println(whisper.SampleBits)

	_ = common.MResolve

	// Starting start/stop server
	if flagHost != "" {
		go func() {
			http.Handle("/", &server{})
			log.Fatal(http.ListenAndServe(flagHost, nil))
		}()
	}

	select {}
}

type server struct{}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/start" && r.URL.Path != "/stop" {
		http.Error(w, "invalid command", 404)
	}

	if flagVerbose {
		log.Printf("Got button request: %v", r.URL.Path)
	}

	defer func() {
		_, _ = w.Write([]byte("OK"))
	}()

	if r.URL.Path == "/start" {
		println("START")
		return
	}

	if r.URL.Path == "/stop" {
		println("STOP")
		return
	}
}
