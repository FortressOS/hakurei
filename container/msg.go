package container

import (
	"errors"
	"log"
	"sync/atomic"
)

// MessageError is an error with a user-facing message.
type MessageError interface {
	// Message returns a user-facing error message.
	Message() string

	error
}

// GetErrorMessage returns whether an error implements [MessageError], and the message if it does.
func GetErrorMessage(err error) (string, bool) {
	var e MessageError
	if !errors.As(err, &e) || e == nil {
		return zeroString, false
	}
	return e.Message(), true
}

// Msg is used for package-wide verbose logging.
type Msg interface {
	// GetLogger returns the address of the underlying [log.Logger].
	GetLogger() *log.Logger

	// IsVerbose atomically loads and returns whether [Msg] has verbose logging enabled.
	IsVerbose() bool
	// SwapVerbose atomically stores a new verbose state and returns the previous value held by [Msg].
	SwapVerbose(verbose bool) bool
	// Verbose passes its argument to the Println method of the underlying [log.Logger] if IsVerbose returns true.
	Verbose(v ...any)
	// Verbosef passes its argument to the Printf method of the underlying [log.Logger] if IsVerbose returns true.
	Verbosef(format string, v ...any)

	// Suspend causes the embedded [Suspendable] to withhold writes to its downstream [io.Writer].
	// Suspend returns false and is a noop if called between calls to Suspend and Resume.
	Suspend() bool

	// Resume dumps the entire buffer held by the embedded [Suspendable] and stops withholding future writes.
	// Resume returns false and is a noop if a call to Suspend does not precede it.
	Resume() bool

	// BeforeExit runs implementation-specific cleanup code, and optionally prints warnings.
	// BeforeExit is called before [os.Exit].
	BeforeExit()
}

// defaultMsg is the default implementation of the [Msg] interface.
// The zero value is not safe for use. Callers should use the [NewMsg] function instead.
type defaultMsg struct {
	verbose atomic.Bool

	logger *log.Logger
	Suspendable
}

// NewMsg initialises a downstream [log.Logger] for a new [Msg].
// The [log.Logger] should no longer be configured after NewMsg returns.
// If downstream is nil, a new logger is initialised in its place.
func NewMsg(downstream *log.Logger) Msg {
	if downstream == nil {
		downstream = log.New(log.Writer(), "container: ", 0)
	}

	m := defaultMsg{logger: downstream}
	m.Suspendable.Downstream = downstream.Writer()
	downstream.SetOutput(&m.Suspendable)
	return &m
}

func (msg *defaultMsg) GetLogger() *log.Logger { return msg.logger }

func (msg *defaultMsg) IsVerbose() bool               { return msg.verbose.Load() }
func (msg *defaultMsg) SwapVerbose(verbose bool) bool { return msg.verbose.Swap(verbose) }
func (msg *defaultMsg) Verbose(v ...any) {
	if msg.verbose.Load() {
		msg.logger.Println(v...)
	}
}
func (msg *defaultMsg) Verbosef(format string, v ...any) {
	if msg.verbose.Load() {
		msg.logger.Printf(format, v...)
	}
}

// Resume calls [Suspendable.Resume] and prints a message if buffer was filled
// between calls to [Suspendable.Suspend] and Resume.
func (msg *defaultMsg) Resume() bool {
	resumed, dropped, _, err := msg.Suspendable.Resume()
	if err != nil {
		// probably going to result in an error as well, so this message is as good as unreachable
		msg.logger.Printf("cannot dump buffer on resume: %v", err)
	}
	if resumed && dropped > 0 {
		msg.logger.Printf("dropped %d bytes while output is suspended", dropped)
	}
	return resumed
}

// BeforeExit prints a message if called between calls to [Suspendable.Suspend] and Resume.
func (msg *defaultMsg) BeforeExit() {
	if msg.Resume() {
		msg.logger.Printf("beforeExit reached on suspended output")
	}
}
