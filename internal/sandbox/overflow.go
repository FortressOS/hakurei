package sandbox

import (
	"bytes"
	"log"
	"os"
	"strconv"
	"sync"
)

var (
	ofUid  int
	ofGid  int
	ofOnce sync.Once
)

const (
	ofUidPath = "/proc/sys/kernel/overflowuid"
	ofGidPath = "/proc/sys/kernel/overflowgid"
)

func mustReadOverflow() {
	if v, err := os.ReadFile(ofUidPath); err != nil {
		log.Fatalf("cannot read %q: %v", ofUidPath, err)
	} else if ofUid, err = strconv.Atoi(string(bytes.TrimSpace(v))); err != nil {
		log.Fatalf("cannot interpret %q: %v", ofUidPath, err)
	}

	if v, err := os.ReadFile(ofGidPath); err != nil {
		log.Fatalf("cannot read %q: %v", ofGidPath, err)
	} else if ofGid, err = strconv.Atoi(string(bytes.TrimSpace(v))); err != nil {
		log.Fatalf("cannot interpret %q: %v", ofGidPath, err)
	}
}

func OverflowUid() int { ofOnce.Do(mustReadOverflow); return ofUid }
func OverflowGid() int { ofOnce.Do(mustReadOverflow); return ofGid }
