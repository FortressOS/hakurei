package internal

import (
	"os"

	"git.gensokyo.uk/security/hakurei/internal/hlog"
)

func Exit(code int) { hlog.BeforeExit(); os.Exit(code) }
