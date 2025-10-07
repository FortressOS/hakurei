package container

import (
	"bytes"
	"os"
	"strconv"
	"sync"

	"hakurei.app/container/fhs"
)

var (
	kernelOverflowuid int
	kernelOverflowgid int
	kernelCapLastCap  int

	sysctlOnce sync.Once
)

const (
	kernelOverflowuidPath = fhs.ProcSys + "kernel/overflowuid"
	kernelOverflowgidPath = fhs.ProcSys + "kernel/overflowgid"
	kernelCapLastCapPath  = fhs.ProcSys + "kernel/cap_last_cap"
)

func mustReadSysctl(msg Msg) {
	sysctlOnce.Do(func() {
		if v, err := os.ReadFile(kernelOverflowuidPath); err != nil {
			msg.GetLogger().Fatalf("cannot read %q: %v", kernelOverflowuidPath, err)
		} else if kernelOverflowuid, err = strconv.Atoi(string(bytes.TrimSpace(v))); err != nil {
			msg.GetLogger().Fatalf("cannot interpret %q: %v", kernelOverflowuidPath, err)
		}

		if v, err := os.ReadFile(kernelOverflowgidPath); err != nil {
			msg.GetLogger().Fatalf("cannot read %q: %v", kernelOverflowgidPath, err)
		} else if kernelOverflowgid, err = strconv.Atoi(string(bytes.TrimSpace(v))); err != nil {
			msg.GetLogger().Fatalf("cannot interpret %q: %v", kernelOverflowgidPath, err)
		}

		if v, err := os.ReadFile(kernelCapLastCapPath); err != nil {
			msg.GetLogger().Fatalf("cannot read %q: %v", kernelCapLastCapPath, err)
		} else if kernelCapLastCap, err = strconv.Atoi(string(bytes.TrimSpace(v))); err != nil {
			msg.GetLogger().Fatalf("cannot interpret %q: %v", kernelCapLastCapPath, err)
		}
	})
}

func OverflowUid(msg Msg) int { mustReadSysctl(msg); return kernelOverflowuid }
func OverflowGid(msg Msg) int { mustReadSysctl(msg); return kernelOverflowgid }
func LastCap(msg Msg) uintptr { mustReadSysctl(msg); return uintptr(kernelCapLastCap) }
