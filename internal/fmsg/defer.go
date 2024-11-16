package fmsg

import (
	"os"
	"sync"
	"sync/atomic"
)

var (
	wstate   atomic.Bool
	dropped  atomic.Uint64
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

// queue submits ops to msgbuf but drops messages
// when the buffer is full and dequeue is withholding
func queue(op dOp) {
	select {
	case msgbuf <- op:
		queueSync.Add(1)
	default:
		// send the op anyway if not withholding
		// as dequeue will get to it eventually
		if !wstate.Load() {
			queueSync.Add(1)
			msgbuf <- op
		} else {
			// increment dropped message count
			dropped.Add(1)
		}
	}
}

type dOp interface{ Do() }

func Exit(code int) {
	queueSync.Wait()
	os.Exit(code)
}

func Suspend() {
	dequeueOnce.Do(dequeue)
	if wstate.CompareAndSwap(false, true) {
		queueSync.Wait()
		withhold <- struct{}{}
	}
}

func Resume() {
	dequeueOnce.Do(dequeue)
	if wstate.CompareAndSwap(true, false) {
		withhold <- struct{}{}
		if d := dropped.Swap(0); d != 0 {
			Printf("dropped %d messages during withhold", d)
		}
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
