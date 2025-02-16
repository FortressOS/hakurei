package internal

import (
	"os"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func Exit(code int) { fmsg.BeforeExit(); os.Exit(code) }
