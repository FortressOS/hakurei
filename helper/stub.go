package helper

import (
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

	// simulate args pipe behaviour
	func() {
		f := os.NewFile(3, "|0")
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
		wait = make(chan struct{})
		go func() {
			f := os.NewFile(4, "|1")
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
		return exec.Command(os.Args[0], append([]string{"-test.run=TestHelperChildStub", "--", name}, arg...)...)
	}
}
