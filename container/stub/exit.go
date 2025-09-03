package stub

import "testing"

// PanicExit is a magic panic value treated as a simulated exit.
const PanicExit = 0xdeadbeef

const (
	panicFailNow = 0xcafe0000 + iota
	panicFatal
	panicFatalf
)

// HandleExit must be deferred before calling with the stub.
func (s *Stub[K]) HandleExit() { handleExit(s.TB, true) }

func handleExit(t testing.TB, root bool) {
	switch r := recover(); r {
	case PanicExit:
		break

	case panicFailNow:
		if root {
			t.FailNow()
		} else {
			t.Fail()
		}
		break

	case panicFatal, panicFatalf, nil:
		break

	default:
		panic(r)
	}
}
