package app

import (
	"os"
	"os/exec"

	"hakurei.app/container/check"
	"hakurei.app/message"
)

type panicDispatcher struct{}

func (panicDispatcher) new(func(k syscallDispatcher))         { panic("unreachable") }
func (panicDispatcher) getpid() int                           { panic("unreachable") }
func (panicDispatcher) getuid() int                           { panic("unreachable") }
func (panicDispatcher) getgid() int                           { panic("unreachable") }
func (panicDispatcher) lookupEnv(string) (string, bool)       { panic("unreachable") }
func (panicDispatcher) stat(string) (os.FileInfo, error)      { panic("unreachable") }
func (panicDispatcher) open(string) (osFile, error)           { panic("unreachable") }
func (panicDispatcher) readdir(string) ([]os.DirEntry, error) { panic("unreachable") }
func (panicDispatcher) tempdir() string                       { panic("unreachable") }
func (panicDispatcher) evalSymlinks(string) (string, error)   { panic("unreachable") }
func (panicDispatcher) lookupGroupId(string) (string, error)  { panic("unreachable") }
func (panicDispatcher) cmdOutput(*exec.Cmd) ([]byte, error)   { panic("unreachable") }
func (panicDispatcher) overflowUid(message.Msg) int           { panic("unreachable") }
func (panicDispatcher) overflowGid(message.Msg) int           { panic("unreachable") }
func (panicDispatcher) mustHsuPath() *check.Absolute          { panic("unreachable") }
func (panicDispatcher) fatalf(string, ...any)                 { panic("unreachable") }
