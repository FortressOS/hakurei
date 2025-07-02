package internal

import (
	"os"

	"hakurei.app/internal/hlog"
)

func Exit(code int) { hlog.BeforeExit(); os.Exit(code) }
