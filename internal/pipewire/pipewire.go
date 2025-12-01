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
	"net"
	"slices"
	"strconv"
	"syscall"
	"time"
)

// Conn is a subset of methods of [net.UnixConn] used by [Context].
type Conn interface {
	// ReadMsgUnix reads a message from c, copying the payload into b and
	// the associated out-of-band data into oob. It returns the number of
	// bytes copied into b, the number of bytes copied into oob, the flags
	// that were set on the message and the source address of the message.
	//
	// Note that if len(b) == 0 and len(oob) > 0, this function will still
	// read (and discard) 1 byte from the connection.
	ReadMsgUnix(b, oob []byte) (n, oobn, flags int, addr *net.UnixAddr, err error)

	// WriteMsgUnix writes a message to addr via c, copying the payload
	// from b and the associated out-of-band data from oob. It returns the
	// number of payload and out-of-band bytes written.
	//
	// Note that if len(b) == 0 and len(oob) > 0, this function will still
	// write 1 byte to the connection.
	WriteMsgUnix(b, oob []byte, addr *net.UnixAddr) (n, oobn int, err error)

	// SetDeadline sets the read and write deadlines associated
	// with the connection. It is equivalent to calling both
	// SetReadDeadline and SetWriteDeadline.
	//
	// A deadline is an absolute time after which I/O operations
	// fail instead of blocking. The deadline applies to all future
	// and pending I/O, not just the immediately following call to
	// Read or Write. After a deadline has been exceeded, the
	// connection can be refreshed by setting a deadline in the future.
	//
	// If the deadline is exceeded a call to Read or Write or to other
	// I/O methods will return an error that wraps os.ErrDeadlineExceeded.
	// This can be tested using errors.Is(err, os.ErrDeadlineExceeded).
	// The error's Timeout method will return true, but note that there
	// are other possible errors for which the Timeout method will
	// return true even if the deadline has not been exceeded.
	//
	// An idle timeout can be implemented by repeatedly extending
	// the deadline after successful Read or Write calls.
	//
	// A zero value for t means I/O operations will not time out.
	SetDeadline(t time.Time) error

	// Close closes the connection.
	// Any blocked Read or Write operations will be unblocked and return errors.
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
	receivedFiles []int
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

	// Passed to [Conn.ReadMsgUnix]. Not copied if sufficient for all received messages.
	iovecBuf [1 << 15]byte
	// Passed to [Conn.ReadMsgUnix] for ancillary messages and is never copied.
	oobBuf [(_SCM_MAX_FD/2+_SCM_MAX_FD%2+2)<<3 + 1]byte
	// Underlying connection, usually implemented by [net.UnixConn].
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

// connTimeout is the maximum duration an I/O operation is allowed for [Conn].
const connTimeout = 5 * time.Second

// receiveAll receives from conn until no more data is available.
// The returned slice is valid until the next call to receiveAll.
func (ctx *Context) receiveAll() (payload []byte, err error) {
	if err = ctx.conn.SetDeadline(time.Now().Add(connTimeout)); err != nil {
		return
	}

	var n, oobn int
	ctx.receivedFiles = ctx.receivedFiles[:0]
	buf := ctx.iovecBuf[:]

recvmsg:
	buf = buf[n:]
	n, oobn, _, _, err = ctx.conn.ReadMsgUnix(buf, ctx.oobBuf[:])
	if err != nil {
		return
	}
	if oobn == len(ctx.oobBuf) {
		return nil, syscall.ENOMEM // unreachable
	}
	if oob := ctx.oobBuf[:oobn]; len(oob) > 0 {
		var scm []syscall.SocketControlMessage
		if scm, err = syscall.ParseSocketControlMessage(oob); err != nil {
			return
		}

		var fds []int
		for i := range scm {
			if fds, err = syscall.ParseUnixRights(&scm[i]); err != nil {
				return
			}
			ctx.receivedFiles = append(ctx.receivedFiles, fds...)
		}
	}

	// receive until buffer fills or payload is depleted
	if n > 0 {
		goto recvmsg
	}
	data := ctx.iovecBuf[:len(ctx.iovecBuf)-len(buf)]

	// avoids copy if payload fits in a single ctx.recvmsgBuf
	if payload == nil && len(buf) > 0 {
		payload = data
		return
	}

	payload = append(payload, data...)
	// this indicates a full ctx.recvmsgBuf
	if len(buf) == 0 {
		ctx.buf = ctx.iovecBuf[:]
		goto recvmsg
	}

	return
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

// RoundtripUnexpectedEOFError is returned when EOF was unexpectedly encountered during [Context.Roundtrip].
type RoundtripUnexpectedEOFError uintptr

const (
	roundtripEOFHeader RoundtripUnexpectedEOFError = iota
	roundtripEOFBody
	roundtripEOFFooter
	roundtripEOFFooterOpcode
)

func (RoundtripUnexpectedEOFError) Unwrap() error { return io.ErrUnexpectedEOF }
func (e RoundtripUnexpectedEOFError) Error() string {
	var suffix string
	switch e {
	case roundtripEOFHeader:
		suffix = "decoding message header"
	case roundtripEOFBody:
		suffix = "establishing message body bounds"
	case roundtripEOFFooter:
		suffix = "establishing message footer bounds"
	case roundtripEOFFooterOpcode:
		suffix = "decoding message footer opcode"

	default:
		return "unexpected EOF"
	}

	return "unexpected EOF " + suffix
}

// eventProxy consumes events during a [Context.Roundtrip].
type eventProxy interface {
	// consume consumes an event and its optional footer.
	consume(opcode byte, files []int, unmarshal func(v any) error) error
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
		return roundtripEOFFooter
	}
	isLastMessage := len(data) == int(header.Size)

	data = data[n:header.Size]
	if len(data) > 0 {
		/* the footer concrete type is determined by opcode, which cannot be
		decoded directly before the type is known, so this hack is required:
		skip the struct prefix, then the integer prefix, and the next SizeId
		bytes are the encoded opcode value */
		if len(data) < int(SizePrefix*2+SizeId) {
			return roundtripEOFFooterOpcode
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
	return "server message headers claim to have sent more than " + strconv.Itoa(int(e)) + " files"
}

// A DanglingFilesError holds onto files that were sent by the server but no [Header]
// accounts for. These must be closed to avoid leaking fds.
type DanglingFilesError []int

func (e DanglingFilesError) Error() string {
	return "received " + strconv.Itoa(len(e)) + " dangling files"
}

// roundtripSyncID is the id passed to Context.coreSync during a [Context.Roundtrip].
const roundtripSyncID = 0

// Roundtrip queues the [CoreSync] message and sends all pending messages to the server.
//
// For a non-nil error, if the error happens over the network, it has concrete type
// [net.OpError].
func (ctx *Context) Roundtrip() (err error) {
	if err = ctx.conn.SetDeadline(time.Now().Add(connTimeout)); err != nil {
		return
	}
	if _, _, err = ctx.conn.WriteMsgUnix(ctx.buf, syscall.UnixRights(ctx.pendingFiles...), nil); err != nil {
		return
	}
	ctx.buf = ctx.buf[:0]
	ctx.pendingFiles = ctx.pendingFiles[:0]
	ctx.headerFiles = 0

	var data []byte
	if data, err = ctx.receiveAll(); err != nil {
		return
	}

	var header Header
	var receivedHeaderFiles int
	for len(data) > 0 {
		if len(data) < SizeHeader {
			return roundtripEOFHeader
		}

		if err = header.UnmarshalBinary(data[:SizeHeader]); err != nil {
			return
		}
		if header.Sequence != ctx.remoteSequence {
			return UnexpectedSequenceError(header.Sequence)
		}
		ctx.remoteSequence++

		if len(data) < int(SizeHeader+header.Size) {
			return roundtripEOFBody
		}

		proxy, ok := ctx.proxy[header.ID]
		if !ok {
			return &UnknownIdError{header.ID, string(data[:SizeHeader+header.Size])}
		}

		nextReceivedHeaderFiles := receivedHeaderFiles + int(header.FileCount)
		if nextReceivedHeaderFiles > len(ctx.receivedFiles) {
			return UnexpectedFilesError(len(ctx.receivedFiles))
		}
		files := ctx.receivedFiles[receivedHeaderFiles:nextReceivedHeaderFiles]
		receivedHeaderFiles = nextReceivedHeaderFiles

		data = data[SizeHeader:]
		err = proxy.consume(header.Opcode, files, func(v any) error { return ctx.unmarshal(&header, data, v) })
		data = data[header.Size:]
		if err != nil {
			return
		}
	}

	if len(ctx.receivedFiles) < receivedHeaderFiles {
		return DanglingFilesError(ctx.receivedFiles[len(ctx.receivedFiles)-receivedHeaderFiles:])
	}
	return
}

// An UnexpectedFileCountError is returned for an event that received an unexpected
// number of files. The proxy closes these extra files before returning
type UnexpectedFileCountError [2]int

func (e *UnexpectedFileCountError) Error() string {
	return "received " + strconv.Itoa(e[1]) + " files instead of the expected " + strconv.Itoa(e[0])
}

// closeReceivedFiles closes all received files and returns [UnexpectedFileCountError]
// if one or more files are passed. This is used with events that do not expect files.
func closeReceivedFiles(fds ...int) error {
	for _, fd := range fds {
		_ = syscall.Close(fd)
	}
	if len(fds) == 0 {
		return nil
	}
	return &UnexpectedFileCountError{0, len(fds)}
}

// Close frees the underlying buffer and closes the connection.
func (ctx *Context) Close() error { ctx.free(); return ctx.conn.Close() }

/* pipewire/device.h */

const (
	PW_TYPE_INTERFACE_Device = PW_TYPE_INFO_INTERFACE_BASE + "Device"
	PW_DEVICE_PERM_MASK      = PW_PERM_RWXM
	PW_VERSION_DEVICE        = 3
)

const (
	PW_DEVICE_CHANGE_MASK_PROPS = 1 << iota
	PW_DEVICE_CHANGE_MASK_PARAMS

	PW_DEVICE_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_DEVICE_EVENT_INFO = iota
	PW_DEVICE_EVENT_PARAM
	PW_DEVICE_EVENT_NUM

	PW_VERSION_DEVICE_EVENTS = 0
)

const (
	PW_DEVICE_METHOD_ADD_LISTENER = iota
	PW_DEVICE_METHOD_SUBSCRIBE_PARAMS
	PW_DEVICE_METHOD_ENUM_PARAMS
	PW_DEVICE_METHOD_SET_PARAM
	PW_DEVICE_METHOD_NUM

	PW_VERSION_DEVICE_METHODS = 0
)

/* pipewire/factory.h */

const (
	PW_TYPE_INTERFACE_Factory = PW_TYPE_INFO_INTERFACE_BASE + "Factory"
	PW_FACTORY_PERM_MASK      = PW_PERM_R | PW_PERM_M
	PW_VERSION_FACTORY        = 3
)

const (
	PW_FACTORY_CHANGE_MASK_PROPS = 1 << iota

	PW_FACTORY_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_FACTORY_EVENT_INFO = iota
	PW_FACTORY_EVENT_NUM

	PW_VERSION_FACTORY_EVENTS = 0
)

const (
	PW_FACTORY_METHOD_ADD_LISTENER = iota
	PW_FACTORY_METHOD_NUM

	PW_VERSION_FACTORY_METHODS = 0
)

/* pipewire/link.h */

const (
	PW_TYPE_INTERFACE_Link = PW_TYPE_INFO_INTERFACE_BASE + "Link"
	PW_LINK_PERM_MASK      = PW_PERM_R | PW_PERM_X
	PW_VERSION_LINK        = 3
)

const (
	PW_LINK_STATE_ERROR       = iota - 2 // the link is in error
	PW_LINK_STATE_UNLINKED               // the link is unlinked
	PW_LINK_STATE_INIT                   // the link is initialized
	PW_LINK_STATE_NEGOTIATING            // the link is negotiating formats
	PW_LINK_STATE_ALLOCATING             // the link is allocating buffers
	PW_LINK_STATE_PAUSED                 // the link is paused
	PW_LINK_STATE_ACTIVE                 // the link is active
)

const (
	PW_LINK_CHANGE_MASK_STATE = (1 << iota)
	PW_LINK_CHANGE_MASK_FORMAT
	PW_LINK_CHANGE_MASK_PROPS

	PW_LINK_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_LINK_EVENT_INFO = iota
	PW_LINK_EVENT_NUM

	PW_VERSION_LINK_EVENTS = 0
)

const (
	PW_LINK_METHOD_ADD_LISTENER = iota
	PW_LINK_METHOD_NUM

	PW_VERSION_LINK_METHODS = 0
)

/* pipewire/module.h */

const (
	PW_TYPE_INTERFACE_Module = PW_TYPE_INFO_INTERFACE_BASE + "Module"
	PW_MODULE_PERM_MASK      = PW_PERM_R | PW_PERM_M
	PW_VERSION_MODULE        = 3
)

const (
	PW_MODULE_CHANGE_MASK_PROPS = 1 << iota

	PW_MODULE_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_MODULE_EVENT_INFO = iota
	PW_MODULE_EVENT_NUM

	PW_VERSION_MODULE_EVENTS = 0
)

const (
	PW_MODULE_METHOD_ADD_LISTENER = iota
	PW_MODULE_METHOD_NUM

	PW_VERSION_MODULE_METHODS = 0
)

/* pipewire/impl-module.h */

const (
	PIPEWIRE_SYMBOL_MODULE_INIT = "pipewire__module_init"
	PIPEWIRE_MODULE_PREFIX      = "libpipewire-"

	PW_VERSION_IMPL_MODULE_EVENTS = 0
)

/* pipewire/node.h */

const (
	PW_TYPE_INTERFACE_Node = PW_TYPE_INFO_INTERFACE_BASE + "Node"
	PW_NODE_PERM_MASK      = PW_PERM_RWXML
	PW_VERSION_NODE        = 3
)

const (
	PW_NODE_STATE_ERROR     = iota - 1 // error state
	PW_NODE_STATE_CREATING             // the node is being created
	PW_NODE_STATE_SUSPENDED            // the node is suspended, the device might be closed
	PW_NODE_STATE_IDLE                 // the node is running but there is no active port
	PW_NODE_STATE_RUNNING              // the node is running
)

const (
	PW_NODE_CHANGE_MASK_INPUT_PORTS = 1 << iota
	PW_NODE_CHANGE_MASK_OUTPUT_PORTS
	PW_NODE_CHANGE_MASK_STATE
	PW_NODE_CHANGE_MASK_PROPS
	PW_NODE_CHANGE_MASK_PARAMS

	PW_NODE_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_NODE_EVENT_INFO = iota
	PW_NODE_EVENT_PARAM
	PW_NODE_EVENT_NUM

	PW_VERSION_NODE_EVENTS = 0
)

const (
	PW_NODE_METHOD_ADD_LISTENER = iota
	PW_NODE_METHOD_SUBSCRIBE_PARAMS
	PW_NODE_METHOD_ENUM_PARAMS
	PW_NODE_METHOD_SET_PARAM
	PW_NODE_METHOD_SEND_COMMAND
	PW_NODE_METHOD_NUM

	PW_VERSION_NODE_METHODS = 0
)

/* pipewire/permission.h */

const (
	PW_PERM_R = 0400 // object can be seen and events can be received
	PW_PERM_W = 0200 // methods can be called that modify the object
	PW_PERM_X = 0100 // methods can be called on the object. The W flag must be present in order to call methods that modify the object.
	PW_PERM_M = 0010 // metadata can be set on object, Since 0.3.9
	PW_PERM_L = 0020 // a link can be made between a node that doesn't have permission to see the other node, Since 0.3.77

	PW_PERM_RW    = PW_PERM_R | PW_PERM_W
	PW_PERM_RWX   = PW_PERM_RW | PW_PERM_X
	PW_PERM_RWXM  = PW_PERM_RWX | PW_PERM_M
	PW_PERM_RWXML = PW_PERM_RWXM | PW_PERM_L

	PW_PERM_ALL          = PW_PERM_RWXM
	PW_PERM_INVALID Word = 0xffffffff
)

/* pipewire/port.h */

const (
	PW_TYPE_INTERFACE_Port = PW_TYPE_INFO_INTERFACE_BASE + "Port"
	PW_PORT_PERM_MASK      = PW_PERM_R | PW_PERM_X | PW_PERM_M
	PW_VERSION_PORT        = 3
)

const (
	PW_PORT_CHANGE_MASK_PROPS = 1 << iota
	PW_PORT_CHANGE_MASK_PARAMS

	PW_PORT_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_PORT_EVENT_INFO = iota
	PW_PORT_EVENT_PARAM
	PW_PORT_EVENT_NUM

	PW_VERSION_PORT_EVENTS = 0
)

const (
	PW_PORT_METHOD_ADD_LISTENER = iota
	PW_PORT_METHOD_SUBSCRIBE_PARAMS
	PW_PORT_METHOD_ENUM_PARAMS
	PW_PORT_METHOD_NUM

	PW_VERSION_PORT_METHODS = 0
)

/* pipewire/extensions/client-node.h */

const (
	PW_TYPE_INTERFACE_ClientNode = PW_TYPE_INFO_INTERFACE_BASE + "ClientNode"
	PW_VERSION_CLIENT_NODE       = 6

	PW_EXTENSION_MODULE_CLIENT_NODE = PIPEWIRE_MODULE_PREFIX + "module-client-node"
)

const (
	PW_CLIENT_NODE_EVENT_TRANSPORT = iota
	PW_CLIENT_NODE_EVENT_SET_PARAM
	PW_CLIENT_NODE_EVENT_SET_IO
	PW_CLIENT_NODE_EVENT_EVENT
	PW_CLIENT_NODE_EVENT_COMMAND
	PW_CLIENT_NODE_EVENT_ADD_PORT
	PW_CLIENT_NODE_EVENT_REMOVE_PORT
	PW_CLIENT_NODE_EVENT_PORT_SET_PARAM
	PW_CLIENT_NODE_EVENT_PORT_USE_BUFFERS
	PW_CLIENT_NODE_EVENT_PORT_SET_IO
	PW_CLIENT_NODE_EVENT_SET_ACTIVATION
	PW_CLIENT_NODE_EVENT_PORT_SET_MIX_INFO
	PW_CLIENT_NODE_EVENT_NUM

	PW_VERSION_CLIENT_NODE_EVENTS = 1
)

const (
	PW_CLIENT_NODE_METHOD_ADD_LISTENER = iota
	PW_CLIENT_NODE_METHOD_GET_NODE
	PW_CLIENT_NODE_METHOD_UPDATE
	PW_CLIENT_NODE_METHOD_PORT_UPDATE
	PW_CLIENT_NODE_METHOD_SET_ACTIVE
	PW_CLIENT_NODE_METHOD_EVENT
	PW_CLIENT_NODE_METHOD_PORT_BUFFERS
	PW_CLIENT_NODE_METHOD_NUM

	PW_VERSION_CLIENT_NODE_METHODS = 0
)

const (
	PW_CLIENT_NODE_UPDATE_PARAMS = 1 << iota
	PW_CLIENT_NODE_UPDATE_INFO
)

const (
	PW_CLIENT_NODE_PORT_UPDATE_PARAMS = 1 << iota
	PW_CLIENT_NODE_PORT_UPDATE_INFO
)

/* pipewire/extensions/metadata.h */

const (
	PW_TYPE_INTERFACE_Metadata = PW_TYPE_INFO_INTERFACE_BASE + "Metadata"
	PW_METADATA_PERM_MASK      = PW_PERM_RWX
	PW_VERSION_METADATA        = 3

	PW_EXTENSION_MODULE_METADATA = PIPEWIRE_MODULE_PREFIX + "module-metadata"
)

const (
	PW_METADATA_EVENT_PROPERTY = iota
	PW_METADATA_EVENT_NUM

	PW_VERSION_METADATA_EVENTS = 0
)

const (
	PW_METADATA_METHOD_ADD_LISTENER = iota
	PW_METADATA_METHOD_SET_PROPERTY
	PW_METADATA_METHOD_CLEAR
	PW_METADATA_METHOD_NUM

	PW_VERSION_METADATA_METHODS = 0
)

const (
	PW_KEY_METADATA_NAME   = "metadata.name"
	PW_KEY_METADATA_VALUES = "metadata.values"
)

/* pipewire/extensions/profiler.h */

const (
	PW_TYPE_INTERFACE_Profiler = PW_TYPE_INFO_INTERFACE_BASE + "Profiler"
	PW_VERSION_PROFILER        = 3
	PW_PROFILER_PERM_MASK      = PW_PERM_R

	PW_EXTENSION_MODULE_PROFILER = PIPEWIRE_MODULE_PREFIX + "module-profiler"
)

const (
	PW_PROFILER_EVENT_PROFILE = iota
	PW_PROFILER_EVENT_NUM

	PW_VERSION_PROFILER_EVENTS = 0
)

const (
	PW_PROFILER_METHOD_ADD_LISTENER = iota
	PW_PROFILER_METHOD_NUM

	PW_VERSION_PROFILER_METHODS = 0
)

const (
	PW_KEY_PROFILER_NAME = "profiler.name"
)

/* pipewire/type.h */

const (
	PW_TYPE_INFO_BASE = "PipeWire:"

	PW_TYPE_INFO_Object      = PW_TYPE_INFO_BASE + "Object"
	PW_TYPE_INFO_OBJECT_BASE = PW_TYPE_INFO_Object + ":"

	PW_TYPE_INFO_Interface      = PW_TYPE_INFO_BASE + "Interface"
	PW_TYPE_INFO_INTERFACE_BASE = PW_TYPE_INFO_Interface + ":"
)
