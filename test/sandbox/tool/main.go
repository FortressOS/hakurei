package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"hakurei.app/test/sandbox"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("test: ")

	if len(os.Args) < 2 {
		log.Fatal("invalid argument")
	}

	switch os.Args[1] {
	case "filter":
		if len(os.Args) != 4 {
			log.Fatal("invalid argument")
		}

		if pid, err := strconv.Atoi(strings.TrimSpace(os.Args[2])); err != nil {
			log.Fatalf("%s", err)
		} else if pid < 1 {
			log.Fatalf("%d out of range", pid)
		} else {
			sandbox.MustCheckFilter(pid, os.Args[3])
			return
		}

	default:
		(&sandbox.T{FS: os.DirFS("/")}).MustCheckFile(os.Args[1], "/tmp/sandbox-ok")
		return
	}
}
