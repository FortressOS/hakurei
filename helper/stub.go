package helper

import (
	"flag"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
)

// InternalChildStub is an internal function but exported because it is cross-package;
// it is part of the implementation of the helper stub.
func InternalChildStub() {
	// this test mocks the helper process
	if os.Getenv(FortifyHelper) != "1" {
		return
	}

	argsFD := flag.Int("args", -1, "")
	statFD := flag.Int("fd", -1, "")
	_ = flag.CommandLine.Parse(os.Args[4:])

	// simulate args pipe behaviour
	func() {
		if *argsFD == -1 {
			panic("attempted to start helper without passing args pipe fd")
		}

		f := os.NewFile(uintptr(*argsFD), "|0")
		if f == nil {
			panic("attempted to start helper without args pipe")
		}

		if _, err := io.Copy(os.Stdout, f); err != nil {
			panic("cannot read args: " + err.Error())
		}
	}()

	var wait chan struct{}

	// simulate status pipe behaviour
	if os.Getenv(FortifyStatus) == "1" {
		if *statFD == -1 {
			panic("attempted to start helper with status reporting without passing status pipe fd")
		}

		wait = make(chan struct{})
		go func() {
			f := os.NewFile(uintptr(*statFD), "|1")
			if f == nil {
				panic("attempted to start with status reporting without status pipe")
			}

			if _, err := f.Write([]byte{'x'}); err != nil {
				panic("cannot write to status pipe: " + err.Error())
			}

			// wait for status pipe close
			var epoll int
			if fd, err := syscall.EpollCreate1(0); err != nil {
				panic("cannot open epoll fd: " + err.Error())
			} else {
				defer func() {
					if err = syscall.Close(fd); err != nil {
						panic("cannot close epoll fd: " + err.Error())
					}
				}()
				epoll = fd
			}
			if err := syscall.EpollCtl(epoll, syscall.EPOLL_CTL_ADD, int(f.Fd()), &syscall.EpollEvent{}); err != nil {
				panic("cannot add status pipe to epoll: " + err.Error())
			}
			events := make([]syscall.EpollEvent, 1)
			if _, err := syscall.EpollWait(epoll, events, -1); err != nil {
				panic("cannot poll status pipe: " + err.Error())
			}
			if events[0].Events != syscall.EPOLLERR {
				panic(strconv.Itoa(int(events[0].Events)))

			}
			close(wait)
		}()
	}

	if wait != nil {
		<-wait
	}
}

// InternalReplaceExecCommand is an internal function but exported because it is cross-package;
// it is part of the implementation of the helper stub.
func InternalReplaceExecCommand(t *testing.T) {
	t.Cleanup(func() {
		execCommand = exec.Command
	})

	// replace execCommand to have the resulting *exec.Cmd launch TestHelperChildStub
	execCommand = func(name string, arg ...string) *exec.Cmd {
		// pass through nonexistent path
		if name == "/nonexistent" && len(arg) == 0 {
			return exec.Command(name)
		}

		return exec.Command(os.Args[0], append([]string{"-test.run=TestHelperChildStub", "--", name}, arg...)...)
	}
}
