package stub

// PanicExit is a magic panic value treated as a simulated exit.
const PanicExit = 0xdeadbeef

// HandleExit must be deferred before calling with the stub.
func HandleExit() {
	r := recover()
	if r == PanicExit {
		return
	}
	if r != nil {
		panic(r)
	}
}
