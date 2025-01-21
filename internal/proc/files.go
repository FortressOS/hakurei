package proc

import (
	"os"
	"os/exec"
)

func ExtraFile(cmd *exec.Cmd, f *os.File) (fd uintptr) {
	return ExtraFileSlice(&cmd.ExtraFiles, f)
}

func ExtraFileSlice(extraFiles *[]*os.File, f *os.File) (fd uintptr) {
	// ExtraFiles: If non-nil, entry i becomes file descriptor 3+i.
	fd = uintptr(3 + len(*extraFiles))
	*extraFiles = append(*extraFiles, f)
	return
}
