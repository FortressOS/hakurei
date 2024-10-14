package dbus

import (
	"errors"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"git.ophivana.moe/cat/fortify/helper"
	"git.ophivana.moe/cat/fortify/helper/bwrap"
	"git.ophivana.moe/cat/fortify/ldd"
)

// Start launches the D-Bus proxy and sets up the Wait method.
// ready should be buffered and must only be received from once.
func (p *Proxy) Start(ready chan error, output io.Writer, sandbox bool) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.seal == nil {
		return errors.New("proxy not sealed")
	}

	var (
		h   helper.Helper
		cmd *exec.Cmd

		argF = func(argsFD, statFD int) []string {
			if statFD == -1 {
				return []string{"--args=" + strconv.Itoa(argsFD)}
			} else {
				return []string{"--args=" + strconv.Itoa(argsFD), "--fd=" + strconv.Itoa(statFD)}
			}
		}
	)

	if !sandbox {
		h = helper.New(p.seal, p.name, argF)
		cmd = h.Unwrap()
		// xdg-dbus-proxy does not need to inherit the environment
		cmd.Env = []string{}
	} else {
		// look up absolute path if name is just a file name
		toolPath := p.name
		if filepath.Base(p.name) == p.name {
			if s, err := exec.LookPath(p.name); err == nil {
				toolPath = s
			}
		}

		// resolve libraries by parsing ldd output
		var proxyDeps []*ldd.Entry
		if path.IsAbs(toolPath) {
			if l, err := ldd.Exec(toolPath); err != nil {
				return err
			} else {
				proxyDeps = l
			}
		}

		bc := &bwrap.Config{
			Unshare:       nil,
			Hostname:      "fortify-dbus",
			Chdir:         "/",
			Clearenv:      true,
			NewSession:    true,
			DieWithParent: true,
		}

		// resolve proxy socket directories
		bindTarget := make(map[string]struct{}, 2)
		for _, ps := range []string{p.session[1], p.system[1]} {
			if pd := path.Dir(ps); len(pd) > 0 {
				if pd[0] == '/' {
					bindTarget[pd] = struct{}{}
				}
			}
		}
		for k := range bindTarget {
			bc.Bind(k, k, false, true)
		}

		roBindTarget := make(map[string]struct{}, 2+1+len(proxyDeps))

		// xdb-dbus-proxy bin and dependencies
		roBindTarget[path.Dir(toolPath)] = struct{}{}
		for _, ent := range proxyDeps {
			if path.IsAbs(ent.Path) {
				roBindTarget[path.Dir(ent.Path)] = struct{}{}
			}
		}

		// resolve upstream bus directories
		for _, as := range []string{p.session[0], p.system[0]} {
			if len(as) > 0 && strings.HasPrefix(as, "unix:path=/") {
				// leave / intact
				roBindTarget[path.Dir(as[10:])] = struct{}{}
			}
		}

		for k := range roBindTarget {
			bc.Bind(k, k)
		}

		h = helper.MustNewBwrap(bc, p.seal, toolPath, argF)
		cmd = h.Unwrap()
		p.bwrap = bc
	}

	if output != nil {
		cmd.Stdout = output
		cmd.Stderr = output
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
