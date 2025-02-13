module github.com/arribada/insight-360/whisper

go 1.22

require (
	github.com/arribada/insight-360-common v0.0.0-20240909132824-a3b9de2f1204
	github.com/arribada/insight-360/common v0.1.0
	github.com/ggerganov/whisper.cpp/bindings/go v0.0.0-20240409172755-8f253ef3af1c
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.1.0
)

require github.com/go-audio/riff v1.0.0 // indirect

replace (
	github.com/arribada/insight-360/common => ../common
	github.com/ggerganov/whisper.cpp/bindings/go => ./whisper.cpp/bindings/go
)
