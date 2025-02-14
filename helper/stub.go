package helper

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/helper/proc"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

// InternalChildStub is an internal function but exported because it is cross-package;
// it is part of the implementation of the helper stub.
func InternalChildStub() {
	// this test mocks the helper process
	var ap, sp string
	if v, ok := os.LookupEnv(FortifyHelper); !ok {
		return
	} else {
		ap = v
	}
	if v, ok := os.LookupEnv(FortifyStatus); !ok {
		panic(FortifyStatus)
	} else {
		sp = v
	}

	switch os.Args[3] {
	case "bwrap":
		bwrapStub()
	default:
		genericStub(flagRestoreFiles(4, ap, sp))
	}

	fmsg.Exit(0)
}

// InternalReplaceExecCommand is an internal function but exported because it is cross-package;
// it is part of the implementation of the helper stub.
func InternalReplaceExecCommand(t *testing.T) {
	t.Cleanup(func() { commandContext = exec.CommandContext })

	// replace execCommand to have the resulting *exec.Cmd launch TestHelperChildStub
	commandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		// pass through nonexistent path
		if name == "/nonexistent" && len(arg) == 0 {
			return exec.CommandContext(ctx, name)
		}

		return exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=TestHelperChildStub", "--", name}, arg...)...)
	}
}

func newFile(fd int, name, p string) *os.File {
	present := false
	switch p {
	case "0":
	case "1":
		present = true
	default:
		panic(fmt.Sprintf("%s fd has unexpected presence value %q", name, p))
	}

	f := os.NewFile(uintptr(fd), name)
	if !present && f != nil {
		panic(fmt.Sprintf("%s fd set but not present", name))
	}
	if present && f == nil {
		panic(fmt.Sprintf("%s fd preset but unset", name))
	}

	return f
}

func flagRestoreFiles(offset int, ap, sp string) (argsFile, statFile *os.File) {
	argsFd := flag.Int("args", -1, "")
	statFd := flag.Int("fd", -1, "")
	_ = flag.CommandLine.Parse(os.Args[offset:])
	argsFile = newFile(*argsFd, "args", ap)
	statFile = newFile(*statFd, "stat", sp)
	return
}

func genericStub(argsFile, statFile *os.File) {
	if argsFile != nil {
		// this output is checked by parent
		if _, err := io.Copy(os.Stdout, argsFile); err != nil {
			panic("cannot read args: " + err.Error())
		}
	}

	// simulate status pipe behaviour
	if statFile != nil {
		if _, err := statFile.Write([]byte{'x'}); err != nil {
			panic("cannot write to status pipe: " + err.Error())
		}

		done := make(chan struct{})
		go func() {
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
			if err := syscall.EpollCtl(epoll, syscall.EPOLL_CTL_ADD, int(statFile.Fd()), &syscall.EpollEvent{}); err != nil {
				panic("cannot add status pipe to epoll: " + err.Error())
			}
			events := make([]syscall.EpollEvent, 1)
			if _, err := syscall.EpollWait(epoll, events, -1); err != nil {
				panic("cannot poll status pipe: " + err.Error())
			}
			if events[0].Events != syscall.EPOLLERR {
				panic(strconv.Itoa(int(events[0].Events)))

			}
			close(done)
		}()
		<-done
	}
}

func bwrapStub() {
	// the bwrap launcher does not launch with a typical sync fd
	argsFile, _ := flagRestoreFiles(4, "1", "0")

	// test args pipe behaviour
	func() {
		got, want := new(strings.Builder), new(strings.Builder)
		if _, err := io.Copy(got, argsFile); err != nil {
			panic("cannot read bwrap args: " + err.Error())
		}

		// hardcoded bwrap configuration used by test
		sc := &bwrap.Config{
			Net:           true,
			Hostname:      "localhost",
			Chdir:         "/nonexistent",
			Clearenv:      true,
			NewSession:    true,
			DieWithParent: true,
			AsInit:        true,
		}
		if _, err := MustNewCheckedArgs(sc.Args(nil, new(proc.ExtraFilesPre), new([]proc.File))).
			WriteTo(want); err != nil {
			panic("cannot read want: " + err.Error())
		}

		if len(flag.CommandLine.Args()) > 0 && flag.CommandLine.Args()[0] == "crash-test-dummy" && got.String() != want.String() {
			panic("bad bwrap args\ngot: " + got.String() + "\nwant: " + want.String())
		}
	}()

	if err := syscall.Exec(
		os.Args[0],
		append([]string{os.Args[0], "-test.run=TestHelperChildStub", "--"}, flag.CommandLine.Args()...),
		os.Environ()); err != nil {
		panic("cannot start general stub: " + err.Error())
	}
}
