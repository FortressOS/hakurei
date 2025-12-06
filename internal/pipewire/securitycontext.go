package pipewire

import (
	"errors"
	"io"
	"os"
	"syscall"
)

/* pipewire/extensions/security-context.h */

const (
	PW_TYPE_INTERFACE_SecurityContext = PW_TYPE_INFO_INTERFACE_BASE + "SecurityContext"
	PW_SECURITY_CONTEXT_PERM_MASK     = PW_PERM_RWX
	PW_VERSION_SECURITY_CONTEXT       = 3

	PW_EXTENSION_MODULE_SECURITY_CONTEXT = PIPEWIRE_MODULE_PREFIX + "module-security-context"
)

const (
	PW_SECURITY_CONTEXT_EVENT_NUM = iota

	PW_VERSION_SECURITY_CONTEXT_EVENTS = 0
)

const (
	PW_SECURITY_CONTEXT_METHOD_ADD_LISTENER = iota
	PW_SECURITY_CONTEXT_METHOD_CREATE
	PW_SECURITY_CONTEXT_METHOD_NUM

	PW_VERSION_SECURITY_CONTEXT_METHODS = 0
)

// SecurityContextCreate is sent to create a new security context.
//
// Creates a new security context with a socket listening FD.
// PipeWire will accept new client connections on listen_fd.
//
// listen_fd must be ready to accept new connections when this
// request is sent by the client. In other words, the client must
// call bind(2) and listen(2) before sending the FD.
//
// close_fd is a FD closed by the client when PipeWire should stop
// accepting new connections on listen_fd.
//
// PipeWire must continue to accept connections on listen_fd when
// the client which created the security context disconnects.
//
// After sending this request, closing listen_fd and close_fd
// remains the only valid operation on them.
type SecurityContextCreate struct {
	// The offset in the SCM_RIGHTS msg_control message to
	// the fd to listen on for new connections.
	ListenFd Fd
	// The offset in the SCM_RIGHTS msg_control message to
	// the fd used to stop listening.
	CloseFd Fd

	// Extra properties. These will be copied on the client
	// that connects through this context.
	Properties *SPADict `json:"props"`
}

// Size satisfies [KnownSize] with a value computed at runtime.
func (c *SecurityContextCreate) Size() Word {
	return SizePrefix +
		Size(SizeFd) +
		Size(SizeFd) +
		c.Properties.Size()
}

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *SecurityContextCreate) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *SecurityContextCreate) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// SecurityContext holds state of [PW_TYPE_INTERFACE_SecurityContext].
type SecurityContext struct {
	// Proxy id as tracked by [Context].
	ID Int `json:"proxy_id"`
	// Global id as tracked by [Registry].
	GlobalID Int `json:"id"`

	ctx *Context
}

// GetSecurityContext queues a [RegistryBind] message for the PipeWire server
// and returns the address of the newly allocated [SecurityContext].
func (registry *Registry) GetSecurityContext() (securityContext *SecurityContext, err error) {
	securityContext = &SecurityContext{ctx: registry.ctx}
	for globalId, object := range registry.Objects {
		if object.Type == securityContext.String() {
			securityContext.GlobalID = globalId
			securityContext.ID, err = registry.bind(securityContext, securityContext.GlobalID, PW_VERSION_SECURITY_CONTEXT)
			return
		}
	}

	return nil, UnsupportedObjectTypeError(securityContext.String())
}

// Create queues a [SecurityContextCreate] message for the PipeWire server.
func (securityContext *SecurityContext) Create(listenFd, closeFd int, props SPADict) error {
	// queued in reverse based on upstream behaviour, unsure why
	offset := securityContext.ctx.queueFiles(closeFd, listenFd)
	return securityContext.ctx.writeMessage(
		securityContext.ID,
		PW_SECURITY_CONTEXT_METHOD_CREATE,
		&SecurityContextCreate{ListenFd: offset + 1, CloseFd: offset + 0, Properties: &props},
	)
}

// securityContextCloser holds onto resources associated to the security context.
type securityContextCloser struct {
	// Pipe with its write end passed to [SecurityContextCreate.CloseFd].
	closeFds [2]int
	// Pathname the socket was bound to.
	pathname string
}

// Close closes both ends of the pipe.
func (scc *securityContextCloser) Close() error {
	return errors.Join(
		syscall.Close(scc.closeFds[1]),
		syscall.Close(scc.closeFds[0]),
		// there is still technically a TOCTOU here but this is internal
		// and has access to the privileged pipewire socket, so it only
		// receives trusted input (e.g. from cmd/hakurei) anyway
		os.Remove(scc.pathname),
	)
}

// BindAndCreate binds a new socket to the specified pathname and pass it to Create.
// It returns an [io.Closer] corresponding to [SecurityContextCreate.CloseFd].
func (securityContext *SecurityContext) BindAndCreate(pathname string, props SPADict) (io.Closer, error) {
	var scc securityContextCloser

	// ensure pathname is available
	if f, err := os.Create(pathname); err != nil {
		return nil, err
	} else if err = f.Close(); err != nil {
		_ = os.Remove(pathname)
		return nil, err
	} else if err = os.Remove(pathname); err != nil {
		return nil, err
	}
	scc.pathname = pathname

	var listenFd int
	if fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0); err != nil {
		return nil, os.NewSyscallError("socket", err)
	} else {
		securityContext.ctx.cleanup(func() error { return syscall.Close(fd) })
		listenFd = fd
	}
	if err := syscall.Bind(listenFd, &syscall.SockaddrUnix{Name: pathname}); err != nil {
		return nil, os.NewSyscallError("bind", err)
	} else if err = syscall.Listen(listenFd, 0); err != nil {
		return nil, os.NewSyscallError("listen", err)
	}

	if err := syscall.Pipe2(scc.closeFds[0:], syscall.O_CLOEXEC); err != nil {
		_ = os.Remove(pathname)
		return nil, err
	}
	if err := securityContext.Create(listenFd, scc.closeFds[1], props); err != nil {
		_ = scc.Close()
		return nil, err
	}
	return &scc, nil
}

func (securityContext *SecurityContext) consume(opcode byte, files []int, _ func(v any)) error {
	closeReceivedFiles(files...)
	switch opcode {
	// SecurityContext does not receive any events

	default:
		return &UnsupportedOpcodeError{opcode, securityContext.String()}
	}

}

func (securityContext *SecurityContext) setBoundProps(event *CoreBoundProps) error {
	if securityContext.ID != event.ID {
		return &InconsistentIdError{Proxy: securityContext, ID: securityContext.ID, ServerID: event.ID}
	}
	if securityContext.GlobalID != event.GlobalID {
		return &InconsistentIdError{Global: true, Proxy: securityContext, ID: securityContext.GlobalID, ServerID: event.GlobalID}
	}
	return nil
}

func (securityContext *SecurityContext) String() string { return PW_TYPE_INTERFACE_SecurityContext }
