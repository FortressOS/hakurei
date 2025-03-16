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
	"git.gensokyo.uk/security/fortify/internal/sandbox"
	"git.gensokyo.uk/security/fortify/ldd"
	"git.gensokyo.uk/security/fortify/seccomp"
)

// Start launches the D-Bus proxy.
func (p *Proxy) Start(ctx context.Context, output io.Writer, useSandbox bool) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.seal == nil {
		return errors.New("proxy not sealed")
	}

	var h helper.Helper

	c, cancel := context.WithCancelCause(ctx)
	if !useSandbox {
		h = helper.NewDirect(c, p.name, p.seal, true, argF, func(cmd *exec.Cmd) {
			if p.CmdF != nil {
				p.CmdF(cmd)
			}
			if output != nil {
				cmd.Stdout, cmd.Stderr = output, output
			}
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Env = make([]string, 0)
		}, nil)
	} else {
		toolPath := p.name
		if filepath.Base(p.name) == p.name {
			if s, err := exec.LookPath(p.name); err != nil {
				return err
			} else {
				toolPath = s
			}
		}

		var libPaths []string
		if entries, err := ldd.ExecFilter(ctx, p.CommandContext, p.FilterF, toolPath); err != nil {
			return err
		} else {
			libPaths = ldd.Path(entries)
		}

		h = helper.New(
			c, toolPath,
			p.seal, true,
			argF, func(container *sandbox.Container) {
				container.Seccomp |= seccomp.FlagMultiarch
				container.Hostname = "fortify-dbus"
				container.CommandContext = p.CommandContext
				if output != nil {
					container.Stdout, container.Stderr = output, output
				}

				if p.CmdF != nil {
					p.CmdF(container)
				}

				// these lib paths are unpredictable, so mount them first so they cannot cover anything
				for _, name := range libPaths {
					container.Bind(name, name, 0)
				}

				// upstream bus directories
				upstreamPaths := make([]string, 0, 2)
				for _, as := range []string{p.session[0], p.system[0]} {
					if len(as) > 0 && strings.HasPrefix(as, "unix:path=/") {
						// leave / intact
						upstreamPaths = append(upstreamPaths, path.Dir(as[10:]))
					}
				}
				slices.Sort(upstreamPaths)
				upstreamPaths = slices.Compact(upstreamPaths)
				for _, name := range upstreamPaths {
					container.Bind(name, name, 0)
				}

				// parent directories of bind paths
				sockDirPaths := make([]string, 0, 2)
				if d := path.Dir(p.session[1]); path.IsAbs(d) {
					sockDirPaths = append(sockDirPaths, d)
				}
				if d := path.Dir(p.system[1]); path.IsAbs(d) {
					sockDirPaths = append(sockDirPaths, d)
				}
				slices.Sort(sockDirPaths)
				sockDirPaths = slices.Compact(sockDirPaths)
				for _, name := range sockDirPaths {
					container.Bind(name, name, sandbox.BindWritable)
				}

				// xdg-dbus-proxy bin path
				binPath := path.Dir(toolPath)
				container.Bind(binPath, binPath, 0)
			}, nil)
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
