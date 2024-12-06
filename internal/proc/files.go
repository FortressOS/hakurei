package proc

import (
	"os"
	"os/exec"
)

func ExtraFile(cmd *exec.Cmd, f *os.File) (fd uintptr) {
	// ExtraFiles: If non-nil, entry i becomes file descriptor 3+i.
	fd = uintptr(3 + len(cmd.ExtraFiles))
	cmd.ExtraFiles = append(cmd.ExtraFiles, f)
	return
}
