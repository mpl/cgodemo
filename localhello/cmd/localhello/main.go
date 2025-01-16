// Copyright (c) The Arribada initiative.
// Licensed under the MIT License.

package main

import (
	"flag"
	"log"
	"net/http"
)

var (
	flagHost    string
	flagVerbose bool
)

func init() {
	flag.BoolVar(&flagVerbose, "v", false, "be verbose")
	flag.StringVar(&flagHost, "host", ":9091", "host:port on which we receive start/stop messages")
}

const newcode = "with looped selfupdate"

func main() {
	flag.Parse()

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
		return
	}

	if flagVerbose {
		log.Printf("Got button request: %v", r.URL.Path)
	}

	defer func() {
		_, _ = w.Write([]byte("OK"))
	}()

	if r.URL.Path == "/start" {
		println("HELLO", newcode)
		return
	}

	if r.URL.Path == "/stop" {
		println("GOODBYE", newcode)
		return
	}
}
