/*
Package helper runs external helpers and manages their status and args FDs.
*/
package helper

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
)

var (
	ErrStatusFault = errors.New("generic status pipe fault")
	ErrStatusRead  = errors.New("unexpected status response")
)

// Helper wraps *exec.Cmd and manages status and args fd.
// Args is always 3 and status if set is always 4.
type Helper struct {
	lock sync.RWMutex
	args io.WriterTo

	statP [2]*os.File
	argsP [2]*os.File

	ready chan error

	// ExtraFiles specifies additional open files to be inherited by the
	// new process. It does not include standard input, standard output, or
	// standard error. If non-nil, entry i becomes file descriptor 5+i.
	ExtraFiles []*os.File

	*exec.Cmd
}

func (h *Helper) StartNotify(ready chan error) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	// Check for doubled Start calls before we defer failure cleanup. If the prior
	// call to Start succeeded, we don't want to spuriously close its pipes.
	if h.Cmd.Process != nil {
		return errors.New("exec: already started")
	}

	// create pipes
	if pr, pw, err := os.Pipe(); err != nil {
		return err
	} else {
		h.argsP[0], h.argsP[1] = pr, pw
	}
	// create status pipes if ready signal is requested
	if ready != nil {
		if pr, pw, err := os.Pipe(); err != nil {
			return err
		} else {
			h.statP[0], h.statP[1] = pr, pw
		}
	}

	// prepare extra files
	el := len(h.ExtraFiles)
	if ready != nil {
		el += 2
	} else {
		el++
	}
	ef := make([]*os.File, 0, el)
	ef = append(ef, h.argsP[0])
	if ready != nil {
		ef = append(ef, h.statP[1])
	}
	ef = append(ef, h.ExtraFiles...)

	// prepare and start process
	h.Cmd.ExtraFiles = ef
	if err := h.Cmd.Start(); err != nil {
		return err
	}

	statsP, argsP := h.statP[0], h.argsP[1]

	// write arguments and close args pipe
	if _, err := h.args.WriteTo(argsP); err != nil {
		if err1 := h.Cmd.Process.Kill(); err1 != nil {
			panic(err1)
		}
		return err
	} else {
		if err = argsP.Close(); err != nil {
			if err1 := h.Cmd.Process.Kill(); err1 != nil {
				panic(err1)
			}
			return err
		}
	}

	if ready != nil {
		h.ready = ready

		// monitor stat pipe
		go func() {
			n, err := statsP.Read(make([]byte, 1))
			switch n {
			case -1:
				if err1 := h.Cmd.Process.Kill(); err1 != nil {
					panic(err1)
				}
				// ensure error is not nil
				if err == nil {
					err = ErrStatusFault
				}
				ready <- err
			case 0:
				// ensure error is not nil
				if err == nil {
					err = ErrStatusRead
				}
				ready <- err
			case 1:
				ready <- nil
			default:
				panic("unreachable") // unexpected read count
			}
		}()
	}

	return nil
}

func (h *Helper) Wait() error {
	h.lock.RLock()
	defer h.lock.RUnlock()

	if h.Cmd.Process == nil {
		return errors.New("exec: not started")
	}
	if h.Cmd.ProcessState != nil {
		return errors.New("exec: Wait was already called")
	}

	// ensure pipe close
	defer func() {
		if err := h.argsP[0].Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			panic(err)
		}
		if err := h.argsP[1].Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			panic(err)
		}

		if h.ready != nil {
			if err := h.statP[0].Close(); err != nil && !errors.Is(err, os.ErrClosed) {
				panic(err)
			}
			if err := h.statP[1].Close(); err != nil && !errors.Is(err, os.ErrClosed) {
				panic(err)
			}
		}
	}()

	return h.Cmd.Wait()
}

func (h *Helper) Close() error {
	if h.ready == nil {
		panic("attempted to close helper with no status pipe")
	}

	return h.statP[0].Close()
}

func (h *Helper) Start() error {
	return h.StartNotify(nil)
}

func New(wt io.WriterTo, name string, arg ...string) *Helper {
	if wt == nil {
		panic("attempted to create helper with nil argument writer")
	}

	return &Helper{args: wt, Cmd: exec.Command(name, arg...)}
}
