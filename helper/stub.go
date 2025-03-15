package helper

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/helper/proc"
	"git.gensokyo.uk/security/fortify/internal"
)

// InternalHelperStub is an internal function but exported because it is cross-package;
// it is part of the implementation of the helper stub.
func InternalHelperStub() {
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

	if len(os.Args) > 3 && os.Args[3] == "bwrap" {
		bwrapStub()
	} else {
		genericStub(flagRestoreFiles(3, ap, sp))
	}

	internal.Exit(0)
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
			Chdir:         "/proc/nonexistent",
			Clearenv:      true,
			NewSession:    true,
			DieWithParent: true,
			AsInit:        true,
		}

		efp := new(proc.ExtraFilesPre)
		if t, ok := os.LookupEnv("GO_TEST_FORTIFY_BWRAP_STUB_TYPE"); ok {
			switch t {
			case "dbus":
				sc.Net = false
				sc.Hostname = "fortify-dbus"
				sc.Chdir = "/"
				sc.Syscall = &bwrap.SyscallPolicy{DenyDevel: true, Multiarch: true}
				sc.AsInit = false

				bindTarget := []string{"/tmp/fortify.1971/12622d846cc3fe7b4c10359d01f0eb47"}
				slices.Sort(bindTarget)
				for _, name := range bindTarget {
					sc.Bind(name, name, false, true)
				}
				roBindTarget := []string{"/run/user/1971", path.Dir(os.Args[0])}
				slices.Sort(roBindTarget)
				for _, name := range roBindTarget {
					sc.Bind(name, name)
				}

				// manipulate extra files list so fd ends up as 5
				efp.Append()
				efp.Append()
			}
		}

		if _, err := MustNewCheckedArgs(sc.Args(nil, efp, new([]proc.File))).
			WriteTo(want); err != nil {
			panic("cannot read want: " + err.Error())
		}

		if got.String() != want.String() {
			panic("bad bwrap args\ngot: " + got.String() + "\nwant: " + want.String())
		}
	}()

	if err := syscall.Exec(
		flag.CommandLine.Args()[0],
		flag.CommandLine.Args(),
		os.Environ()); err != nil {
		panic("cannot start helper stub: " + err.Error())
	}
}
