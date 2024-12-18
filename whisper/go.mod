module github.com/arribada/insight-360/whisper

go 1.22

require (
	github.com/arribada/insight-360-common v0.0.0-20240909132824-a3b9de2f1204
	github.com/ggerganov/whisper.cpp/bindings/go v0.0.0-20240409172755-8f253ef3af1c
)

replace github.com/ggerganov/whisper.cpp/bindings/go => ./whisper.cpp/bindings/go
