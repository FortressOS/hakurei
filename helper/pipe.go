package helper

import (
	"errors"
	"io"
	"os"
	"os/exec"

	"git.gensokyo.uk/security/fortify/internal/proc"
)

type pipes struct {
	args io.WriterTo

	statP [2]*os.File
	argsP [2]*os.File

	ready chan error

	cmd *exec.Cmd
}

func (p *pipes) pipe() error {
	if p.statP[0] != nil || p.statP[1] != nil ||
		p.argsP[0] != nil || p.argsP[1] != nil {
		panic("attempted to pipe twice")
	}
	if p.args == nil {
		panic("attempted to pipe without args")
	}

	// create pipes
	if pr, pw, err := os.Pipe(); err != nil {
		return err
	} else {
		p.argsP[0], p.argsP[1] = pr, pw
	}

	// create status pipes if ready signal is requested
	if p.ready != nil {
		if pr, pw, err := os.Pipe(); err != nil {
			return err
		} else {
			p.statP[0], p.statP[1] = pr, pw
		}
	}

	return nil
}

// calls pipe to create pipes and sets them up as ExtraFiles, returning their fd
func (p *pipes) prepareCmd(cmd *exec.Cmd) (argsFd, statFd int, err error) {
	argsFd, statFd = -1, -1
	if err = p.pipe(); err != nil {
		return
	}

	// save a reference of cmd for future use
	p.cmd = cmd

	argsFd = int(proc.ExtraFile(cmd, p.argsP[0]))
	if p.ready != nil {
		statFd = int(proc.ExtraFile(cmd, p.statP[1]))
	}

	return
}

func (p *pipes) readyWriteArgs() error {
	statsP, argsP := p.statP[0], p.argsP[1]

	// write arguments and close args pipe
	if _, err := p.args.WriteTo(argsP); err != nil {
		if err1 := p.cmd.Process.Kill(); err1 != nil {
			// should be unreachable
			panic(err1.Error())
		}
		return err
	} else {
		if err = argsP.Close(); err != nil {
			if err1 := p.cmd.Process.Kill(); err1 != nil {
				// should be unreachable
				panic(err1.Error())
			}
			return err
		}
	}

	if p.ready != nil {
		// monitor stat pipe
		go func() {
			n, err := statsP.Read(make([]byte, 1))
			switch n {
			case -1:
				if err1 := p.cmd.Process.Kill(); err1 != nil {
					// should be unreachable
					panic(err1.Error())
				}
				// ensure error is not nil
				if err == nil {
					err = ErrStatusFault
				}
				p.ready <- err
			case 0:
				// ensure error is not nil
				if err == nil {
					err = ErrStatusRead
				}
				p.ready <- err
			case 1:
				p.ready <- nil
			default:
				panic("unreachable") // unexpected read count
			}
		}()
	}

	return nil
}

func (p *pipes) mustClosePipes() {
	if err := p.argsP[0].Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		// unreachable
		panic(err.Error())
	}
	if err := p.argsP[1].Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		// unreachable
		panic(err.Error())
	}

	if p.ready != nil {
		if err := p.statP[0].Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			// unreachable
			panic(err.Error())
		}
		if err := p.statP[1].Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			// unreachable
			panic(err.Error())
		}
	}
}

func (p *pipes) closeStatus() error {
	if p.ready == nil {
		panic("attempted to close helper with no status pipe")
	}

	return p.statP[0].Close()
}
