//go:build testtool

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"hakurei.app/test/internal/sandbox"
)

var (
	flagMarkerPath string
	flagTestCase   string
	flagBpfHash    string
)

func init() {
	flag.StringVar(&flagMarkerPath, "p", "/tmp/sandbox-ok", "Pathname of completion marker")
	flag.StringVar(&flagTestCase, "t", "", "Nix store path to test case file")
	flag.StringVar(&flagBpfHash, "s", "", "String representation of expected bpf sha512 hash")
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("test: ")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		s := make(chan os.Signal, 1)
		signal.Notify(s, syscall.SIGINT)
		go func() { <-s; log.Println("exiting on signal (likely from verifier)"); os.Exit(0) }()

		(&sandbox.T{FS: os.DirFS("/")}).MustCheckFile(flagTestCase)
		if _, err := os.Create(flagMarkerPath); err != nil {
			log.Fatalf("cannot create success marker: %v", err)
		}
		log.Printf("blocking for seccomp check (%s)", flagMarkerPath)
		select {}
		return
	}

	switch args[0] {
	case "filter":
		if len(args) != 2 {
			log.Fatal("invalid argument")
		}

		if pid, err := strconv.Atoi(strings.TrimSpace(args[1])); err != nil {
			log.Fatalf("%s", err)
		} else if pid < 1 {
			log.Fatalf("%d out of range", pid)
		} else {
			sandbox.MustCheckFilter(pid, flagBpfHash)
			if err = syscall.Kill(pid, syscall.SIGINT); err != nil {
				log.Fatalf("cannot signal check process: %v", err)
			}
		}

	case "hash": // this eases the pain of passing the hash to python
		fmt.Print(flagBpfHash)

	case "fd":
		if len(args) != 2 {
			log.Fatal("invalid argument")
		}
		prefix := fmt.Sprintf("/proc/%s/fd/", args[1])

		var fail bool
		if entries, err := os.ReadDir(prefix); err != nil {
			log.Fatal(err.Error())
		} else {
			for _, ent := range entries {
				var fd int
				if fd, err = strconv.Atoi(ent.Name()); err != nil {
					log.Fatal(err.Error())
				}

				// skip standard streams
				if fd <= 2 {
					continue
				}
				fail = true

				var d string
				if d, err = os.Readlink(prefix + ent.Name()); err != nil {
					log.Fatal(err.Error())
				}
				log.Printf("[FAIL] extra fd %d -> %s", fd, d)
			}
		}
		if fail {
			log.Fatal("[FAIL] file descriptors leaked")
		}

	default:
		log.Fatal("invalid argument")
	}
}
