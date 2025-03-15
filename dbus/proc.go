package dbus

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"git.gensokyo.uk/security/fortify/helper"
	"git.gensokyo.uk/security/fortify/helper/bwrap"
	"git.gensokyo.uk/security/fortify/ldd"
)

// Start launches the D-Bus proxy.
func (p *Proxy) Start(ctx context.Context, output io.Writer, sandbox bool) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.seal == nil {
		return errors.New("proxy not sealed")
	}

	var h helper.Helper

	c, cancel := context.WithCancelCause(ctx)
	if !sandbox {
		h = helper.NewDirect(c, p.name, p.seal, true, argF, func(cmd *exec.Cmd) {
			cmdF(cmd, output, p.CmdF)

			// xdg-dbus-proxy does not need to inherit the environment
			cmd.Env = make([]string, 0)
		}, nil)
	} else {
		// look up absolute path if name is just a file name
		toolPath := p.name
		if filepath.Base(p.name) == p.name {
			if s, err := exec.LookPath(p.name); err != nil {
				return err
			} else {
				toolPath = s
			}
		}

		// resolve libraries by parsing ldd output
		var proxyDeps []*ldd.Entry
		if toolPath != os.Args[0] {
			if l, err := ldd.Exec(ctx, toolPath); err != nil {
				return err
			} else {
				proxyDeps = l
			}
		}

		bc := &bwrap.Config{
			Hostname:      "fortify-dbus",
			Chdir:         "/",
			Syscall:       &bwrap.SyscallPolicy{DenyDevel: true, Multiarch: true},
			Clearenv:      true,
			NewSession:    true,
			DieWithParent: true,
		}

		// resolve proxy socket directories
		bindTargetM := make(map[string]struct{}, 2)

		for _, ps := range []string{p.session[1], p.system[1]} {
			if pd := path.Dir(ps); len(pd) > 0 {
				if pd[0] == '/' {
					bindTargetM[pd] = struct{}{}
				}
			}
		}

		bindTarget := make([]string, 0, len(bindTargetM))
		for k := range bindTargetM {
			bindTarget = append(bindTarget, k)
		}
		slices.Sort(bindTarget)
		for _, name := range bindTarget {
			bc.Bind(name, name, false, true)
		}

		roBindTargetM := make(map[string]struct{}, 2+1+len(proxyDeps))

		// xdb-dbus-proxy bin and dependencies
		roBindTargetM[path.Dir(toolPath)] = struct{}{}
		for _, ent := range proxyDeps {
			if path.IsAbs(ent.Path) {
				roBindTargetM[path.Dir(ent.Path)] = struct{}{}
			}
			if path.IsAbs(ent.Name) {
				roBindTargetM[path.Dir(ent.Name)] = struct{}{}
			}
		}

		// resolve upstream bus directories
		for _, as := range []string{p.session[0], p.system[0]} {
			if len(as) > 0 && strings.HasPrefix(as, "unix:path=/") {
				// leave / intact
				roBindTargetM[path.Dir(as[10:])] = struct{}{}
			}
		}

		roBindTarget := make([]string, 0, len(roBindTargetM))
		for k := range roBindTargetM {
			roBindTarget = append(roBindTarget, k)
		}
		slices.Sort(roBindTarget)
		for _, name := range roBindTarget {
			bc.Bind(name, name)
		}

		h = helper.MustNewBwrap(c, toolPath,
			p.seal, true,
			argF, func(cmd *exec.Cmd) { cmdF(cmd, output, p.CmdF) },
			nil,
			bc, nil,
		)
		p.bwrap = bc
	}

	if err := h.Start(); err != nil {
		cancel(err)
		return err
	}

	p.helper = h
	p.ctx = c
	p.cancel = cancel
	return nil
}

var proxyClosed = errors.New("proxy closed")

// Wait blocks until xdg-dbus-proxy exits and releases resources.
func (p *Proxy) Wait() error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.helper == nil {
		return errors.New("dbus: not started")
	}

	errs := make([]error, 3)

	errs[0] = p.helper.Wait()
	if p.cancel == nil &&
		errors.Is(errs[0], context.Canceled) &&
		errors.Is(context.Cause(p.ctx), proxyClosed) {
		errs[0] = nil
	}

	// ensure socket removal so ephemeral directory is empty at revert
	if err := os.Remove(p.session[1]); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs[1] = err
	}
	if p.sysP {
		if err := os.Remove(p.system[1]); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs[2] = err
		}
	}

	return errors.Join(errs...)
}

// Close cancels the context passed to the helper instance attached to xdg-dbus-proxy.
func (p *Proxy) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.cancel == nil {
		panic("dbus: not started")
	}
	p.cancel(proxyClosed)
	p.cancel = nil
}

func argF(argsFd, statFd int) []string {
	if statFd == -1 {
		return []string{"--args=" + strconv.Itoa(argsFd)}
	} else {
		return []string{"--args=" + strconv.Itoa(argsFd), "--fd=" + strconv.Itoa(statFd)}
	}
}

func cmdF(cmd *exec.Cmd, output io.Writer, cmdF func(cmd *exec.Cmd)) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if output != nil {
		cmd.Stdout, cmd.Stderr = output, output
	}
	if cmdF != nil {
		cmdF(cmd)
	}
}
