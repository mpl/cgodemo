package main

import (
	"os"
)

type capHeader struct {
	version uint32
	pid     int
}

type capData struct {
	effective   uint32
	permitted   uint32
	inheritable uint32
}

type caps struct {
	hdr  capHeader
	data [2]capData
}

func getCaps() (caps, error) {
	var c caps
	return c, nil
}

// mustDropPrivileges executes the program in a child process, dropping root
// privileges, but retaining the CAP_SYS_TIME capability to change the system
// clock.
func mustDropPrivileges(rtc *os.File) {}
