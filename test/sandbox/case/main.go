package main

import (
	"os"

	"git.gensokyo.uk/security/fortify/test/sandbox"
)

func main() { (&sandbox.T{FS: os.DirFS("/")}).MustCheckFile(os.Args[1], "/tmp/sandbox-ok") }
