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
func HandleExit(t testing.TB) {
	switch r := recover(); r {
	case PanicExit:
		break

	case panicFailNow:
		t.FailNow()

	case panicFatal, panicFatalf, nil:
		break

	default:
		panic(r)
	}
}

// handleExitNew handles exits from goroutines created by [Stub.New].
func handleExitNew(t testing.TB) {
	switch r := recover(); r {
	case PanicExit, panicFatal, panicFatalf, nil:
		break

	case panicFailNow:
		t.Fail()
		break

	default:
		panic(r)
	}
}
