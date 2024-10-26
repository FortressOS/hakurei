package fmsg

import (
	"os"
	"sync"
	"sync/atomic"
)

var (
	wstate   atomic.Bool
	withhold = make(chan struct{}, 1)
	msgbuf   = make(chan dOp, 64) // these ops are tiny so a large buffer is allocated for withholding output

	dequeueOnce sync.Once
	queueSync   sync.WaitGroup
)

func dequeue() {
	go func() {
		for {
			select {
			case op := <-msgbuf:
				op.Do()
				queueSync.Done()
			case <-withhold:
				<-withhold
			}
		}
	}()
}

type dOp interface{ Do() }

func Exit(code int) {
	queueSync.Wait()
	os.Exit(code)
}

func Withhold() {
	if wstate.CompareAndSwap(false, true) {
		withhold <- struct{}{}
	}
}

func Resume() {
	if wstate.CompareAndSwap(true, false) {
		withhold <- struct{}{}
	}
}

type dPrint []any

func (v dPrint) Do() {
	std.Print(v...)
}

type dPrintf struct {
	format string
	v      []any
}

func (d *dPrintf) Do() {
	std.Printf(d.format, d.v...)
}

type dPrintln []any

func (v dPrintln) Do() {
	std.Println(v...)
}
