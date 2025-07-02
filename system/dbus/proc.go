package dbus

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"

	"git.gensokyo.uk/security/hakurei/container"
	"git.gensokyo.uk/security/hakurei/container/seccomp"
	"git.gensokyo.uk/security/hakurei/helper"
	"git.gensokyo.uk/security/hakurei/ldd"
)

// Start starts and configures a D-Bus proxy process.
func (p *Proxy) Start() error {
	if p.final == nil || p.final.WriterTo == nil {
		return syscall.ENOTRECOVERABLE
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.pmu.Lock()
	defer p.pmu.Unlock()

	if p.cancel != nil || p.cause != nil {
		return errors.New("dbus: already started")
	}

	ctx, cancel := context.WithCancelCause(p.ctx)

	if !p.useSandbox {
		p.helper = helper.NewDirect(ctx, p.name, p.final, true, argF, func(cmd *exec.Cmd) {
			if p.CmdF != nil {
				p.CmdF(cmd)
			}
			if p.output != nil {
				cmd.Stdout, cmd.Stderr = p.output, p.output
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

		p.helper = helper.New(
			ctx, toolPath,
			p.final, true,
			argF, func(z *container.Container) {
				z.SeccompFlags |= seccomp.AllowMultiarch
				z.SeccompPresets |= seccomp.PresetStrict
				z.Hostname = "hakurei-dbus"
				z.CommandContext = p.CommandContext
				if p.output != nil {
					z.Stdout, z.Stderr = p.output, p.output
				}

				if p.CmdF != nil {
					p.CmdF(z)
				}

				// these lib paths are unpredictable, so mount them first so they cannot cover anything
				for _, name := range libPaths {
					z.Bind(name, name, 0)
				}

				// upstream bus directories
				upstreamPaths := make([]string, 0, 2)
				for _, addr := range [][]AddrEntry{p.final.SessionUpstream, p.final.SystemUpstream} {
					for _, ent := range addr {
						if ent.Method != "unix" {
							continue
						}
						for _, pair := range ent.Values {
							if pair[0] != "path" || !path.IsAbs(pair[1]) {
								continue
							}
							upstreamPaths = append(upstreamPaths, path.Dir(pair[1]))
						}
					}
				}
				slices.Sort(upstreamPaths)
				upstreamPaths = slices.Compact(upstreamPaths)
				for _, name := range upstreamPaths {
					z.Bind(name, name, 0)
				}

				// parent directories of bind paths
				sockDirPaths := make([]string, 0, 2)
				if d := path.Dir(p.final.Session[1]); path.IsAbs(d) {
					sockDirPaths = append(sockDirPaths, d)
				}
				if d := path.Dir(p.final.System[1]); path.IsAbs(d) {
					sockDirPaths = append(sockDirPaths, d)
				}
				slices.Sort(sockDirPaths)
				sockDirPaths = slices.Compact(sockDirPaths)
				for _, name := range sockDirPaths {
					z.Bind(name, name, container.BindWritable)
				}

				// xdg-dbus-proxy bin path
				binPath := path.Dir(toolPath)
				z.Bind(binPath, binPath, 0)
			}, nil)
	}

	if err := p.helper.Start(); err != nil {
		cancel(err)
		p.helper = nil
		return err
	}

	p.cancel, p.cause = cancel, func() error { return context.Cause(ctx) }
	return nil
}

var proxyClosed = errors.New("proxy closed")

// Wait blocks until xdg-dbus-proxy exits and releases resources.
func (p *Proxy) Wait() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.pmu.RLock()
	if p.helper == nil || p.cancel == nil || p.cause == nil {
		p.pmu.RUnlock()
		return errors.New("dbus: not started")
	}

	errs := make([]error, 3)

	errs[0] = p.helper.Wait()
	if errors.Is(errs[0], context.Canceled) &&
		errors.Is(p.cause(), proxyClosed) {
		errs[0] = nil
	}
	p.pmu.RUnlock()

	// ensure socket removal so ephemeral directory is empty at revert
	if err := os.Remove(p.final.Session[1]); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs[1] = err
	}
	if p.final.System[1] != "" {
		if err := os.Remove(p.final.System[1]); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs[2] = err
		}
	}

	return errors.Join(errs...)
}

// Close cancels the context passed to the helper instance attached to xdg-dbus-proxy.
func (p *Proxy) Close() {
	p.pmu.Lock()
	defer p.pmu.Unlock()

	if p.cancel == nil {
		panic("dbus: not started")
	}
	p.cancel(proxyClosed)
}

func argF(argsFd, statFd int) []string {
	if statFd == -1 {
		return []string{"--args=" + strconv.Itoa(argsFd)}
	} else {
		return []string{"--args=" + strconv.Itoa(argsFd), "--fd=" + strconv.Itoa(statFd)}
	}
}
