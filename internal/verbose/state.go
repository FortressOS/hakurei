package verbose

import "sync/atomic"

var verbose = new(atomic.Bool)

func Get() bool {
	return verbose.Load()
}

func Set(v bool) {
	verbose.Store(v)
}
