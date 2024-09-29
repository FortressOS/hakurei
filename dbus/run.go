package dbus

import (
	"errors"
	"io"

	"git.ophivana.moe/cat/fortify/helper"
)

// Start launches the D-Bus proxy and sets up the Wait method.
// ready should be buffered and should only be received from once.
func (p *Proxy) Start(ready chan error, output io.Writer) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.seal == nil {
		return errors.New("proxy not sealed")
	}

	h := helper.New(p.seal, p.path,
		// Helper: Args is always 3 and status if set is always 4.
		"--args=3",
		"--fd=4",
	)
	// xdg-dbus-proxy does not need to inherit the environment
	h.Env = []string{}

	if output != nil {
		h.Stdout = output
		h.Stderr = output
	}
	if err := h.StartNotify(ready); err != nil {
		return err
	}

	p.helper = h
	return nil
}

// Wait waits for xdg-dbus-proxy to exit or fault.
func (p *Proxy) Wait() error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.helper == nil {
		return errors.New("proxy not started")
	}

	return p.helper.Wait()
}

// Close closes the status file descriptor passed to xdg-dbus-proxy, causing it to stop.
func (p *Proxy) Close() error {
	return p.helper.Close()
}
