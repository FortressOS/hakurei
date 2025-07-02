package container

import (
	"bytes"
	"log"
	"os"
	"strconv"
	"sync"
)

var (
	kernelOverflowuid int
	kernelOverflowgid int
	kernelCapLastCap  int

	sysctlOnce sync.Once
)

const (
	kernelOverflowuidPath = "/proc/sys/kernel/overflowuid"
	kernelOverflowgidPath = "/proc/sys/kernel/overflowgid"
	kernelCapLastCapPath  = "/proc/sys/kernel/cap_last_cap"
)

func mustReadSysctl() {
	if v, err := os.ReadFile(kernelOverflowuidPath); err != nil {
		log.Fatalf("cannot read %q: %v", kernelOverflowuidPath, err)
	} else if kernelOverflowuid, err = strconv.Atoi(string(bytes.TrimSpace(v))); err != nil {
		log.Fatalf("cannot interpret %q: %v", kernelOverflowuidPath, err)
	}

	if v, err := os.ReadFile(kernelOverflowgidPath); err != nil {
		log.Fatalf("cannot read %q: %v", kernelOverflowgidPath, err)
	} else if kernelOverflowgid, err = strconv.Atoi(string(bytes.TrimSpace(v))); err != nil {
		log.Fatalf("cannot interpret %q: %v", kernelOverflowgidPath, err)
	}

	if v, err := os.ReadFile(kernelCapLastCapPath); err != nil {
		log.Fatalf("cannot read %q: %v", kernelCapLastCapPath, err)
	} else if kernelCapLastCap, err = strconv.Atoi(string(bytes.TrimSpace(v))); err != nil {
		log.Fatalf("cannot interpret %q: %v", kernelCapLastCapPath, err)
	}
}

func OverflowUid() int { sysctlOnce.Do(mustReadSysctl); return kernelOverflowuid }
func OverflowGid() int { sysctlOnce.Do(mustReadSysctl); return kernelOverflowgid }
func LastCap() uintptr { sysctlOnce.Do(mustReadSysctl); return uintptr(kernelCapLastCap) }
