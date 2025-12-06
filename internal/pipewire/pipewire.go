// Package pipewire provides a partial implementation of the PipeWire protocol native.
//
// This implementation is created based on black box analysis and very limited static
// analysis. The PipeWire documentation is vague and mostly nonexistent, and source code
// readability is not great due to frequent macro abuse, confusing and inconsistent naming
// schemes, almost complete absence of comments and the multiple layers of abstractions
// even internal to the library. The convoluted build system and frequent (mis)use of
// dlopen(3) further complicates static analysis efforts.
//
// Because of this, extreme care must be taken when reusing any code found in this package.
// While it is extensively tested to be correct for its role within Hakurei, remember that
// work is only done against PipeWire behaviour specific to this use case, and it is nearly
// impossible to guarantee that this interpretation of its behaviour is intended, or correct
// for any other uses of the protocol.
package pipewire

import (
	"encoding/binary"
	"fmt"
	"io"
	"maps"
	"os"
	"runtime"
	"slices"
	"strconv"
	"syscall"
)

// Conn is a low level unix socket interface used by [Context].
type Conn interface {
	// Recvmsg calls syscall.Recvmsg on the underlying socket.
	Recvmsg(p, oob []byte, flags int) (n, oobn, recvflags int, err error)

	// Sendmsg calls syscall.SendmsgN on the underlying socket.
	Sendmsg(p, oob []byte, flags int) (n int, err error)

	// Close closes the connection.
	Close() error
}

// The kernel constant SCM_MAX_FD defines a limit on the number of file descriptors in the array.
// Attempting to send an array larger than this limit causes sendmsg(2) to fail with the error
// EINVAL. SCM_MAX_FD has the value 253 (or 255 before Linux 2.6.38).
const _SCM_MAX_FD = 253

// A Context holds state of a connection to PipeWire.
type Context struct {
	// Pending message data, committed via a call to Roundtrip.
	buf []byte
	// Current [Header.Sequence] value, incremented every write.
	sequence Int
	// Current server-side [Header.Sequence] value, incremented on every event processed.
	remoteSequence Int
	// Proxy id associations.
	proxy map[Int]eventProxy
	// Newly allocated proxies pending acknowledgement from the server.
	pendingIds map[Int]struct{}
	// Smallest available Id for the next proxy.
	nextId Int
	// Server side registry generation number.
	generation Long
	// Pending file descriptors to be sent with the next message.
	pendingFiles []int
	// File count kept track of in [Header].
	headerFiles int
	// Files from the server. This is discarded on every Roundtrip so eventProxy
	// implementations must make sure to close them to avoid leaking fds.
	//
	// These are not automatically set up as [os.File] because it is impossible
	// to undo the effects of os.NewFile, which can be inconvenient for some uses.
	receivedFiles []int
	// Non-protocol errors encountered during event handling of the current Roundtrip;
	// errors that prevent event processing from continuing must be panicked.
	proxyErrors ProxyConsumeError
	// Pending footer value for the next outgoing message.
	// Newer footers appear to simply replace the existing one.
	pendingFooter KnownSize
	// Pending footer value deferred to the next round trip,
	// sent if pendingFooter is nil. This is for emulating upstream behaviour
	deferredPendingFooter KnownSize
	// Proxy for built-in core events.
	core Core
	// Proxy for built-in client events.
	client Client

	// Passed to [Conn.Recvmsg]. Not copied if sufficient for all received messages.
	iovecBuf [1 << 15]byte
	// Passed to [Conn.Recvmsg] for ancillary messages and is never copied.
	oobBuf [(_SCM_MAX_FD/2+_SCM_MAX_FD%2+2)<<3 + 1]byte
	// Underlying connection, usually implemented by [net.UnixConn]
	// via the [SyscallConn] adapter.
	conn Conn
}

// GetCore returns the address of [Core] held by this [Context].
func (ctx *Context) GetCore() *Core { return &ctx.core }

// GetClient returns the address of [Client] held by this [Context].
func (ctx *Context) GetClient() *Client { return &ctx.client }

// New initialises [Context] for an already established connection and returns its address.
// The caller must not call any method of the underlying [Conn] after this function returns.
func New(conn Conn, props SPADict) (*Context, error) {
	ctx := Context{conn: conn}
	ctx.core.ctx = &ctx
	ctx.proxy = map[Int]eventProxy{
		PW_ID_CORE:   &ctx.core,
		PW_ID_CLIENT: &ctx.client,
	}
	ctx.pendingIds = map[Int]struct{}{
		PW_ID_CLIENT: {},
	}
	ctx.nextId = Int(len(ctx.proxy))

	if err := ctx.coreHello(); err != nil {
		return nil, err
	}
	if err := ctx.clientUpdateProperties(props); err != nil {
		return nil, err
	}

	return &ctx, nil
}

// A SyscallConnCloser is a [syscall.Conn] that implements [io.Closer].
type SyscallConnCloser interface {
	syscall.Conn
	io.Closer
}

// A SyscallConn is a [Conn] adapter for [syscall.Conn].
type SyscallConn struct{ SyscallConnCloser }

// Recvmsg implements [Conn.Recvmsg] via [syscall.Conn.SyscallConn].
func (conn SyscallConn) Recvmsg(p, oob []byte, flags int) (n, oobn, recvflags int, err error) {
	var rc syscall.RawConn
	if rc, err = conn.SyscallConn(); err != nil {
		return
	}

	if controlErr := rc.Control(func(fd uintptr) {
		n, oobn, recvflags, _, err = syscall.Recvmsg(int(fd), p, oob, flags)
	}); controlErr != nil && err == nil {
		err = controlErr
	}
	return
}

// Sendmsg implements [Conn.Sendmsg] via [syscall.Conn.SyscallConn].
func (conn SyscallConn) Sendmsg(p, oob []byte, flags int) (n int, err error) {
	var rc syscall.RawConn
	if rc, err = conn.SyscallConn(); err != nil {
		return
	}

	if controlErr := rc.Control(func(fd uintptr) {
		n, err = syscall.SendmsgN(int(fd), p, oob, nil, flags)
	}); controlErr != nil && err == nil {
		err = controlErr
	}
	return
}

// MustNew calls [New](conn, props) and panics on error.
// It is intended for use in tests with hard-coded strings.
func MustNew(conn Conn, props SPADict) *Context {
	if ctx, err := New(conn, props); err != nil {
		panic(err)
	} else {
		return ctx
	}
}

// free releases the underlying storage of buf.
func (ctx *Context) free() { ctx.buf = make([]byte, 0) }

// queueFiles queues some file descriptors to be sent for the next message.
// It returns the offset of their index for the syscall.SCM_RIGHTS message.
func (ctx *Context) queueFiles(fds ...int) (offset Fd) {
	offset = Fd(len(ctx.pendingFiles))
	ctx.pendingFiles = append(ctx.pendingFiles, fds...)
	return
}

// writeMessage appends the POD representation of v and an optional footer to buf.
func (ctx *Context) writeMessage(
	Id Int, opcode byte,
	v KnownSize,
) (err error) {
	if ctx.pendingFooter == nil && ctx.deferredPendingFooter != nil {
		ctx.pendingFooter, ctx.deferredPendingFooter = ctx.deferredPendingFooter, nil
	}

	size := v.Size()
	if ctx.pendingFooter != nil {
		size += ctx.pendingFooter.Size()
	}
	if size&^SizeMax != 0 {
		return ErrSizeRange
	}

	ctx.buf = slices.Grow(ctx.buf, int(SizeHeader+size))
	ctx.buf = (&Header{
		ID: Id, Opcode: opcode, Size: size,
		Sequence:  ctx.sequence,
		FileCount: Int(len(ctx.pendingFiles) - ctx.headerFiles),
	}).append(ctx.buf)
	ctx.headerFiles = len(ctx.pendingFiles)
	ctx.buf, err = MarshalAppend(ctx.buf, v)
	if err == nil && ctx.pendingFooter != nil {
		ctx.buf, err = MarshalAppend(ctx.buf, ctx.pendingFooter)
		ctx.pendingFooter = nil
	}
	ctx.sequence++
	return
}

// newProxyId returns a newly allocated proxy Id for the specified type.
func (ctx *Context) newProxyId(proxy eventProxy, ack bool) Int {
	newId := ctx.nextId
	ctx.proxy[newId] = proxy
	if ack {
		ctx.pendingIds[newId] = struct{}{}
	}

increment:
	ctx.nextId++

	if _, ok := ctx.proxy[ctx.nextId]; ok {
		goto increment
	}
	return newId
}

// closeReceivedFiles closes all receivedFiles. This is only during protocol error
// where [Context] is rendered unusable.
func (ctx *Context) closeReceivedFiles() {
	slices.Sort(ctx.receivedFiles)
	ctx.receivedFiles = slices.Compact(ctx.receivedFiles)
	for _, fd := range ctx.receivedFiles {
		_ = syscall.Close(fd)
	}
	ctx.receivedFiles = ctx.receivedFiles[:0]
}

// recvmsgFlags are flags passed to [Conn.Recvmsg] during Context.recvmsg.
const recvmsgFlags = syscall.MSG_CMSG_CLOEXEC | syscall.MSG_DONTWAIT

// recvmsg receives from conn and returns the received payload backed by
// iovecBuf. The returned slice is valid until the next call to recvmsg.
func (ctx *Context) recvmsg(remaining []byte) (payload []byte, err error) {
	if copy(ctx.iovecBuf[:], remaining) != len(remaining) {
		// should not be reachable with correct internal usage
		return remaining, syscall.ENOMEM
	}

	var n, oobn, recvflags int

	for {
		n, oobn, recvflags, err = ctx.conn.Recvmsg(ctx.iovecBuf[len(remaining):], ctx.oobBuf[:], recvmsgFlags)

		if oob := ctx.oobBuf[:oobn]; len(oob) > 0 {
			var scm []syscall.SocketControlMessage
			if scm, err = syscall.ParseSocketControlMessage(oob); err != nil {
				ctx.closeReceivedFiles()
				return
			}

			var fds []int
			for i := range scm {
				if fds, err = syscall.ParseUnixRights(&scm[i]); err != nil {
					ctx.closeReceivedFiles()
					return
				}
				ctx.receivedFiles = append(ctx.receivedFiles, fds...)
			}
		}

		if recvflags&syscall.MSG_CTRUNC != 0 {
			// unreachable
			ctx.closeReceivedFiles()
			return nil, syscall.ENOMEM
		}

		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if err != syscall.EAGAIN && err != syscall.EWOULDBLOCK {
				ctx.closeReceivedFiles()
				return nil, os.NewSyscallError("recvmsg", err)
			}
		}

		break
	}

	if n == 0 && len(remaining) != len(ctx.iovecBuf) && err == nil {
		err = syscall.EPIPE // not wrapped as it did not come from the syscall
	}
	if n > 0 {
		payload = ctx.iovecBuf[:n]
	}
	return
}

// sendmsgFlags are flags passed to [Conn.Sendmsg] during Context.sendmsg.
const sendmsgFlags = syscall.MSG_NOSIGNAL | syscall.MSG_DONTWAIT

// sendmsg sends p to conn. sendmsg does not retain p.
func (ctx *Context) sendmsg(p []byte, fds ...int) error {
	var oob []byte
	if len(fds) > 0 {
		oob = syscall.UnixRights(fds...)
	}

	for {
		n, err := ctx.conn.Sendmsg(p, oob, sendmsgFlags)
		if err == syscall.EINTR {
			continue
		}

		if err == nil && n != len(p) {
			err = syscall.EMSGSIZE
		}

		if err != nil && err != syscall.EAGAIN && err != syscall.EWOULDBLOCK {
			return os.NewSyscallError("sendmsg", err)
		}
		return err
	}
}

// An UnknownIdError describes a server message with an Id unknown to [Context].
type UnknownIdError struct {
	// Offending id decoded from Data.
	Id Int
	// Message received from the server.
	Data string
}

func (e *UnknownIdError) Error() string { return "unknown proxy id " + strconv.Itoa(int(e.Id)) }

// UnsupportedOpcodeError describes a message with an unsupported opcode.
type UnsupportedOpcodeError struct {
	// Offending opcode.
	Opcode byte
	// Name of interface processed by the proxy.
	Interface string
}

func (e *UnsupportedOpcodeError) Error() string {
	return "unsupported " + e.Interface + " opcode " + strconv.Itoa(int(e.Opcode))
}

// UnsupportedFooterOpcodeError describes a [Footer] with an unsupported opcode.
type UnsupportedFooterOpcodeError Id

func (e UnsupportedFooterOpcodeError) Error() string {
	return "unsupported footer opcode " + strconv.Itoa(int(e))
}

// A RoundtripUnexpectedEOFError describes an unexpected EOF encountered during [Context.Roundtrip].
type RoundtripUnexpectedEOFError uintptr

const (
	// ErrRoundtripEOFHeader is returned when unexpectedly encountering EOF
	// decoding the message header.
	ErrRoundtripEOFHeader RoundtripUnexpectedEOFError = iota
	// ErrRoundtripEOFBody is returned when unexpectedly encountering EOF
	// establishing message body bounds.
	ErrRoundtripEOFBody
	// ErrRoundtripEOFFooter is like [ErrRoundtripEOFBody], but for when establishing
	// bounds for the footer instead.
	ErrRoundtripEOFFooter
	// ErrRoundtripEOFFooterOpcode is returned when unexpectedly encountering EOF
	// during the footer opcode hack.
	ErrRoundtripEOFFooterOpcode
)

func (RoundtripUnexpectedEOFError) Unwrap() error { return io.ErrUnexpectedEOF }
func (e RoundtripUnexpectedEOFError) Error() string {
	var suffix string
	switch e {
	case ErrRoundtripEOFHeader:
		suffix = "decoding message header"
	case ErrRoundtripEOFBody:
		suffix = "establishing message body bounds"
	case ErrRoundtripEOFFooter:
		suffix = "establishing message footer bounds"
	case ErrRoundtripEOFFooterOpcode:
		suffix = "decoding message footer opcode"

	default:
		return "unexpected EOF"
	}

	return "unexpected EOF " + suffix
}

// eventProxy consumes events during a [Context.Roundtrip].
type eventProxy interface {
	// consume consumes an event and its optional footer.
	consume(opcode byte, files []int, unmarshal func(v any)) error
	// setBoundProps stores a [CoreBoundProps] event received from the server.
	setBoundProps(event *CoreBoundProps) error

	// Stringer returns the PipeWire interface name.
	fmt.Stringer
}

// unmarshal is like [Unmarshal] but handles footer if present.
func (ctx *Context) unmarshal(header *Header, data []byte, v any) error {
	n, err := UnmarshalNext(data, v)
	if err != nil {
		return err
	}
	if len(data) < int(header.Size) || header.Size < n {
		return ErrRoundtripEOFFooter
	}
	isLastMessage := len(data) == int(header.Size)

	data = data[n:header.Size]
	if len(data) > 0 {
		/* the footer concrete type is determined by opcode, which cannot be
		decoded directly before the type is known, so this hack is required:
		skip the struct prefix, then the integer prefix, and the next SizeId
		bytes are the encoded opcode value */
		if len(data) < int(SizePrefix*2+SizeId) {
			return ErrRoundtripEOFFooterOpcode
		}
		switch opcode := binary.NativeEndian.Uint32(data[SizePrefix*2:]); opcode {
		case FOOTER_CORE_OPCODE_GENERATION:
			var footer Footer[FooterCoreGeneration]
			if err = Unmarshal(data, &footer); err != nil {
				return err
			}
			if ctx.generation != footer.Payload.RegistryGeneration {
				var pendingFooter = Footer[FooterClientGeneration]{
					FOOTER_CORE_OPCODE_GENERATION,
					FooterClientGeneration{ClientGeneration: footer.Payload.RegistryGeneration},
				}

				// this emulates upstream behaviour that pending footer updated on the last message
				// during a roundtrip is pushed back to the first message of the next roundtrip
				if isLastMessage {
					ctx.deferredPendingFooter = &pendingFooter
				} else {
					ctx.pendingFooter = &pendingFooter
				}
			}
			ctx.generation = footer.Payload.RegistryGeneration
			return nil

		default:
			return UnsupportedFooterOpcodeError(opcode)
		}
	}
	return nil
}

// An UnexpectedSequenceError is a server-side sequence number that does not
// match its counterpart tracked by the client. This indicates that either
// the client has somehow missed events, or data being interpreted as [Header]
// is, in fact, not the message header.
type UnexpectedSequenceError Int

func (e UnexpectedSequenceError) Error() string { return "unexpected seq " + strconv.Itoa(int(e)) }

// An UnexpectedFilesError describes an inconsistent state where file count claimed by
// [Header] accumulates to a value greater than the total number of files received.
type UnexpectedFilesError int

func (e UnexpectedFilesError) Error() string {
	return "server message headers claim to have sent more files than actually received"
}

// A DanglingFilesError holds onto files that were sent by the server but no [Header]
// accounts for. These are closed by their finalizers if discarded.
type DanglingFilesError []*os.File

func (e DanglingFilesError) Error() string {
	return "received " + strconv.Itoa(len(e)) + " dangling files"
}

// An UnacknowledgedProxyError holds newly allocated proxy ids that the server failed
// to acknowledge after an otherwise successful [Context.Roundtrip].
type UnacknowledgedProxyError []Int

func (e UnacknowledgedProxyError) Error() string {
	return "server did not acknowledge " + strconv.Itoa(len(e)) + " proxies"
}

// A ProxyFatalError describes an error that terminates event handling during a
// [Context.Roundtrip] and makes further event processing no longer possible.
type ProxyFatalError struct {
	// The fatal error causing the termination of event processing.
	Err error
	// Previous non-fatal proxy errors.
	ProxyErrs []error
}

func (e *ProxyFatalError) Unwrap() []error { return append(e.ProxyErrs, e.Err) }
func (e *ProxyFatalError) Error() string {
	s := e.Err.Error()
	if len(e.ProxyErrs) > 0 {
		s += "; " + strconv.Itoa(len(e.ProxyErrs)) + " additional proxy errors occurred before this point"
	}
	return s
}

// A ProxyConsumeError is a collection of non-protocol errors returned by proxies
// during event processing. These do not prevent event handling from continuing but
// may be considered fatal to the application.
type ProxyConsumeError []error

func (e ProxyConsumeError) Unwrap() []error { return e }
func (e ProxyConsumeError) Error() string {
	if len(e) == 0 {
		return "invalid proxy consume error"
	}

	// first error is usually the most relevant one
	s := e[0].Error()
	if len(e) > 1 {
		s += "; " + strconv.Itoa(len(e)) + " additional proxy errors occurred after this point"
	}
	return s
}

// cloneAsProxyErrors clones and truncates proxyErrors if it contains errors,
// returning the cloned slice.
func (ctx *Context) cloneAsProxyErrors() (proxyErrors ProxyConsumeError) {
	if len(ctx.proxyErrors) == 0 {
		return
	}
	proxyErrors = slices.Clone(ctx.proxyErrors)
	ctx.proxyErrors = ctx.proxyErrors[:0]
	return
}

// cloneProxyErrors is like cloneAsProxyErrors, but returns nil if proxyErrors
// does not contain errors.
func (ctx *Context) cloneProxyErrors() (err error) {
	proxyErrors := ctx.cloneAsProxyErrors()
	if len(proxyErrors) > 0 {
		err = proxyErrors
	}
	return
}

// roundtripSyncID is the id passed to Context.coreSync during a [Context.Roundtrip].
const roundtripSyncID = 0

// Roundtrip sends all pending messages to the server and processes events until
// the server has no more messages.
//
// For a non-nil error, if the error happens over the network, it has concrete type
// [os.SyscallError].
func (ctx *Context) Roundtrip() (err error) {
	err = ctx.roundtrip()
	if err == nil {
		err = ctx.cloneProxyErrors()
	}
	return
}

// roundtrip implements the Roundtrip method without checking proxyErrors.
func (ctx *Context) roundtrip() (err error) {
	if err = ctx.sendmsg(ctx.buf, ctx.pendingFiles...); err != nil {
		return
	}

	var remaining []byte
	for {
		remaining, err = ctx.consume(remaining)
		if err == nil {
			continue
		}

		// only returned by recvmsg
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			if len(remaining) == 0 {
				err = nil
			} else if len(remaining) < SizeHeader {
				err = &ProxyFatalError{Err: ErrRoundtripEOFHeader, ProxyErrs: ctx.cloneAsProxyErrors()}
			} else {
				err = &ProxyFatalError{Err: ErrRoundtripEOFBody, ProxyErrs: ctx.cloneAsProxyErrors()}
			}
		}
		return
	}
}

// consume receives messages from the server and processes events.
func (ctx *Context) consume(receiveRemaining []byte) (remaining []byte, err error) {
	defer func() {
		// anything before this has already been processed and must not be closed
		// here, as anything holding onto them will end up with a dangling fd that
		// can be reused and cause serious problems
		if len(ctx.receivedFiles) > 0 {
			ctx.closeReceivedFiles()

			// this catches cases where Roundtrip somehow returns without processing
			// all received files or preparing an error for dangling files, this is
			// always overwritten by the fatal error being processed below or made
			// inaccessible due to repanicking, so if this ends up returned to the
			// caller it indicates something has gone seriously wrong in Roundtrip
			if err == nil {
				err = syscall.ENOTRECOVERABLE
			}
		}

		r := recover()
		if r == nil {
			return
		}

		recoveredErr, ok := r.(error)
		if !ok {
			panic(r)
		}
		if recoveredErr == nil {
			panic(&runtime.PanicNilError{})
		}

		err = &ProxyFatalError{Err: recoveredErr, ProxyErrs: ctx.cloneAsProxyErrors()}
		return
	}()

	ctx.buf = ctx.buf[:0]
	ctx.pendingFiles = ctx.pendingFiles[:0]
	ctx.headerFiles = 0

	if remaining, err = ctx.recvmsg(receiveRemaining); err != nil {
		return
	}

	var header Header
	for len(remaining) > 0 {
		if len(remaining) < SizeHeader {
			return
		}

		if err = header.UnmarshalBinary(remaining[:SizeHeader]); err != nil {
			return
		}
		if header.Sequence != ctx.remoteSequence {
			return remaining, UnexpectedSequenceError(header.Sequence)
		}
		ctx.remoteSequence++

		if len(remaining) < int(SizeHeader+header.Size) {
			return
		}

		proxy, ok := ctx.proxy[header.ID]
		if !ok {
			return remaining, &UnknownIdError{header.ID, string(remaining[:SizeHeader+header.Size])}
		}

		fileCount := int(header.FileCount)
		if fileCount > len(ctx.receivedFiles) {
			return remaining, UnexpectedFilesError(fileCount)
		}
		files := ctx.receivedFiles[:fileCount]
		ctx.receivedFiles = ctx.receivedFiles[fileCount:]

		remaining = remaining[SizeHeader:]
		proxyErr := proxy.consume(header.Opcode, files, func(v any) {
			if unmarshalErr := ctx.unmarshal(&header, remaining, v); unmarshalErr != nil {
				panic(unmarshalErr)
			}
		})
		remaining = remaining[header.Size:]
		if proxyErr != nil {
			ctx.proxyErrors = append(ctx.proxyErrors, proxyErr)
		}
	}

	// prepared here so finalizers are set up, but should not prevent proxyErrors
	// from reaching the caller as those describe the cause of these dangling fds
	var danglingFiles DanglingFilesError
	if len(ctx.receivedFiles) > 0 {
		// having multiple *os.File with the same fd causes serious problems
		slices.Sort(ctx.receivedFiles)
		ctx.receivedFiles = slices.Compact(ctx.receivedFiles)

		danglingFiles = make(DanglingFilesError, 0, len(ctx.receivedFiles))
		for _, fd := range ctx.receivedFiles {
			// hold these as *os.File so they are closed if this error never reaches the caller,
			// or the caller discards or otherwise does not handle this error, to avoid leaking fds
			danglingFiles = append(danglingFiles, os.NewFile(uintptr(fd),
				"dangling fd "+strconv.Itoa(fd)+" received from PipeWire"))
		}
		ctx.receivedFiles = ctx.receivedFiles[:0]
	}

	// populated early for finalizers
	if len(danglingFiles) > 0 {
		return remaining, danglingFiles
	}

	// this check must happen after everything else passes
	if len(ctx.pendingIds) != 0 {
		return remaining, UnacknowledgedProxyError(slices.Collect(maps.Keys(ctx.pendingIds)))
	}
	return
}

// An UnexpectedFileCountError is returned as part of a [ProxyFatalError] for an event
// that received an unexpected number of files.
type UnexpectedFileCountError [2]int

func (e *UnexpectedFileCountError) Error() string {
	return "received " + strconv.Itoa(e[1]) + " files instead of the expected " + strconv.Itoa(e[0])
}

// closeReceivedFiles closes all received files and panics with [UnexpectedFileCountError]
// if one or more files are passed. This is used with events that do not expect files.
func closeReceivedFiles(fds ...int) {
	for _, fd := range fds {
		_ = syscall.Close(fd)
	}
	if len(fds) > 0 {
		panic(&UnexpectedFileCountError{0, len(fds)})
	}
}

// Close frees the underlying buffer and closes the connection.
func (ctx *Context) Close() error { ctx.free(); return ctx.conn.Close() }
