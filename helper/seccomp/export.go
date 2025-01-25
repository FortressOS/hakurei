package seccomp

import (
	"io"
	"os"
)

func Export(opts SyscallOpts) (f *os.File, err error) {
	if f, err = tmpfile(); err != nil {
		return
	}
	if err = exportFilter(f.Fd(), opts); err != nil {
		return
	}
	_, err = f.Seek(0, io.SeekStart)
	return
}
