package dbus

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"hakurei.app/container"
	"hakurei.app/container/check"
	"hakurei.app/container/seccomp"
	"hakurei.app/container/std"
	"hakurei.app/helper"
	"hakurei.app/ldd"
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
			if p.output != nil {
				cmd.Stdout, cmd.Stderr = p.output, p.output
			}
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Env = make([]string, 0)
		}, nil)
	} else {
		var toolPath *check.Absolute
		if a, err := check.NewAbs(p.name); err != nil {
			if p.name, err = exec.LookPath(p.name); err != nil {
				return err
			} else if toolPath, err = check.NewAbs(p.name); err != nil {
				return err
			}
		} else {
			toolPath = a
		}

		var libPaths []*check.Absolute
		if entries, err := ldd.Exec(ctx, p.msg, toolPath.String()); err != nil {
			return err
		} else {
			libPaths = ldd.Path(entries)
		}

		p.helper = helper.New(
			ctx, p.msg, toolPath, "xdg-dbus-proxy",
			p.final, true,
			argF, func(z *container.Container) {
				z.SeccompFlags |= seccomp.AllowMultiarch
				z.SeccompPresets |= std.PresetStrict
				z.Hostname = "hakurei-dbus"
				if p.output != nil {
					z.Stdout, z.Stderr = p.output, p.output
				}

				// these lib paths are unpredictable, so mount them first so they cannot cover anything
				for _, name := range libPaths {
					z.Bind(name, name, 0)
				}

				// upstream bus directories
				upstreamPaths := make([]*check.Absolute, 0, 2)
				for _, addr := range [][]AddrEntry{p.final.SessionUpstream, p.final.SystemUpstream} {
					for _, ent := range addr {
						if ent.Method != "unix" {
							continue
						}
						for _, pair := range ent.Values {
							if pair[0] != "path" {
								continue
							}
							if a, err := check.NewAbs(pair[1]); err != nil {
								continue
							} else {
								upstreamPaths = append(upstreamPaths, a.Dir())
							}
						}
					}
				}
				check.SortAbs(upstreamPaths)
				upstreamPaths = check.CompactAbs(upstreamPaths)
				for _, name := range upstreamPaths {
					z.Bind(name, name, 0)
				}
				z.HostNet = len(upstreamPaths) == 0
				z.HostAbstract = z.HostNet

				// parent directories of bind paths
				sockDirPaths := make([]*check.Absolute, 0, 2)
				if a, err := check.NewAbs(p.final.Session[1]); err == nil {
					sockDirPaths = append(sockDirPaths, a.Dir())
				}
				if a, err := check.NewAbs(p.final.System[1]); err == nil {
					sockDirPaths = append(sockDirPaths, a.Dir())
				}
				check.SortAbs(sockDirPaths)
				sockDirPaths = check.CompactAbs(sockDirPaths)
				for _, name := range sockDirPaths {
					z.Bind(name, name, std.BindWritable)
				}

				// xdg-dbus-proxy bin path
				binPath := toolPath.Dir()
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

	var errs [3]error

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

	return errors.Join(errs[:]...)
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
