package dbus

import (
	"errors"
	"os"
	"os/exec"
)

// Start launches the D-Bus proxy and sets up the Wait method.
// ready should be buffered and should only be received from once.
func (p *Proxy) Start(ready *chan bool) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.seal == nil {
		return errors.New("proxy not sealed")
	}

	// acquire pipes
	if pr, pw, err := os.Pipe(); err != nil {
		return err
	} else {
		p.statP[0], p.statP[1] = pr, pw
	}
	if pr, pw, err := os.Pipe(); err != nil {
		return err
	} else {
		p.argsP[0], p.argsP[1] = pr, pw
	}

	p.cmd = exec.Command(p.path,
		// ExtraFiles: If non-nil, entry i becomes file descriptor 3+i.
		"--fd=3",
		"--args=4",
	)
	p.cmd.Env = []string{}
	p.cmd.ExtraFiles = []*os.File{p.statP[1], p.argsP[0]}
	p.cmd.Stdout = os.Stdout
	p.cmd.Stderr = os.Stderr
	if err := p.cmd.Start(); err != nil {
		return err
	}

	statsP, argsP := p.statP[0], p.argsP[1]

	if _, err := argsP.Write([]byte(*p.seal)); err != nil {
		if err1 := p.cmd.Process.Kill(); err1 != nil {
			panic(err1)
		}
		return err
	} else {
		if err = argsP.Close(); err != nil {
			if err1 := p.cmd.Process.Kill(); err1 != nil {
				panic(err1)
			}
			return err
		}
	}

	wait := make(chan error)
	go func() {
		// live out the lifespan of the process
		wait <- p.cmd.Wait()
	}()

	read := make(chan error)
	go func() {
		n, err := statsP.Read(make([]byte, 1))
		switch n {
		case -1:
			if err1 := p.cmd.Process.Kill(); err1 != nil {
				panic(err1)
			}
			read <- err
		case 0:
			read <- err
		case 1:
			*ready <- true
			read <- nil
		default:
			panic("unreachable") // unexpected read count
		}
	}()

	p.wait = &wait
	p.read = &read
	p.ready = ready

	return nil
}

// Wait waits for xdg-dbus-proxy to exit or fault.
func (p *Proxy) Wait() error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.wait == nil || p.read == nil {
		return errors.New("proxy not running")
	}

	defer func() {
		if err1 := p.statP[0].Close(); err1 != nil && !errors.Is(err1, os.ErrClosed) {
			panic(err1)
		}
		if err1 := p.statP[1].Close(); err1 != nil && !errors.Is(err1, os.ErrClosed) {
			panic(err1)
		}

		if err1 := p.argsP[0].Close(); err1 != nil && !errors.Is(err1, os.ErrClosed) {
			panic(err1)
		}
		if err1 := p.argsP[1].Close(); err1 != nil && !errors.Is(err1, os.ErrClosed) {
			panic(err1)
		}

	}()

	select {
	case err := <-*p.wait:
		*p.ready <- false
		return err
	case err := <-*p.read:
		if err != nil {
			*p.ready <- false
			return err
		}
		return <-*p.wait
	}
}

// Close closes the status file descriptor passed to xdg-dbus-proxy, causing it to stop.
func (p *Proxy) Close() error {
	return p.statP[0].Close()
}
