package pipewire

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

/* pipewire/core.h */

const (
	PW_TYPE_INTERFACE_Core     = PW_TYPE_INFO_INTERFACE_BASE + "Core"
	PW_TYPE_INTERFACE_Registry = PW_TYPE_INFO_INTERFACE_BASE + "Registry"
	PW_CORE_PERM_MASK          = PW_PERM_R | PW_PERM_X | PW_PERM_M
	PW_VERSION_CORE            = 4
	PW_VERSION_REGISTRY        = 3

	PW_DEFAULT_REMOTE = "pipewire-0"
	PW_ID_CORE        = 0
	PW_ID_ANY         = Word(0xffffffff)
)

const (
	PW_CORE_CHANGE_MASK_PROPS = 1 << iota

	PW_CORE_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_CORE_EVENT_INFO = iota
	PW_CORE_EVENT_DONE
	PW_CORE_EVENT_PING
	PW_CORE_EVENT_ERROR
	PW_CORE_EVENT_REMOVE_ID
	PW_CORE_EVENT_BOUND_ID
	PW_CORE_EVENT_ADD_MEM
	PW_CORE_EVENT_REMOVE_MEM
	PW_CORE_EVENT_BOUND_PROPS
	PW_CORE_EVENT_NUM

	PW_VERSION_CORE_EVENTS = 1
)

const (
	PW_CORE_METHOD_ADD_LISTENER = iota
	PW_CORE_METHOD_HELLO
	PW_CORE_METHOD_SYNC
	PW_CORE_METHOD_PONG
	PW_CORE_METHOD_ERROR
	PW_CORE_METHOD_GET_REGISTRY
	PW_CORE_METHOD_CREATE_OBJECT
	PW_CORE_METHOD_DESTROY
	PW_CORE_METHOD_NUM

	PW_VERSION_CORE_METHODS = 0
)

const (
	PW_REGISTRY_EVENT_GLOBAL = iota
	PW_REGISTRY_EVENT_GLOBAL_REMOVE
	PW_REGISTRY_EVENT_NUM

	PW_VERSION_REGISTRY_EVENTS = 0
)

const (
	PW_REGISTRY_METHOD_ADD_LISTENER = iota
	PW_REGISTRY_METHOD_BIND
	PW_REGISTRY_METHOD_DESTROY
	PW_REGISTRY_METHOD_NUM

	PW_VERSION_REGISTRY_METHODS = 0
)

const (
	FOOTER_CORE_OPCODE_GENERATION = iota

	FOOTER_CORE_OPCODE_LAST
)

// The FooterCoreGeneration indicates to the client what is the current
// registry generation number of the Context on the server side.
//
// The server shall include this footer in the next message it sends that
// follows the increment of the registry generation number.
type FooterCoreGeneration struct {
	RegistryGeneration Long `json:"registry_generation"`
}

// Size satisfies [KnownSize] with a constant value.
func (fcg FooterCoreGeneration) Size() Word {
	return SizePrefix +
		Size(SizeLong)
}

// The FooterClientGeneration indicates to the server what is the last
// registry generation number the client has processed.
//
// The client shall include this footer in the next message it sends,
// after it has processed an incoming message whose footer includes a
// registry generation update.
type FooterClientGeneration struct {
	ClientGeneration Long `json:"client_generation"`
}

// Size satisfies [KnownSize] with a constant value.
func (fcg FooterClientGeneration) Size() Word {
	return SizePrefix +
		Size(SizeLong)
}

// A CoreInfo event is emitted by the server upon connection
// with the more information about the server.
type CoreInfo struct {
	// The id of the server (PW_ID_CORE).
	ID Int `json:"id"`
	// A unique cookie for this server.
	Cookie Int `json:"cookie"`
	// The name of the user running the server.
	UserName String `json:"user_name"`
	// The name of the host running the server.
	HostName String `json:"host_name"`
	// A version string of the server.
	Version String `json:"version"`
	// The name of the server.
	Name String `json:"name"`
	// A set of bits with changes to the info.
	ChangeMask Long `json:"change_mask"`
	// Optional key/value properties, valid when change_mask has PW_CORE_CHANGE_MASK_PROPS.
	Properties *SPADict `json:"props"`
}

// Size satisfies [KnownSize] with a value computed at runtime.
func (c *CoreInfo) Size() Word {
	return SizePrefix +
		Size(SizeInt) +
		Size(SizeInt) +
		SizeString[Word](c.UserName) +
		SizeString[Word](c.HostName) +
		SizeString[Word](c.Version) +
		SizeString[Word](c.Name) +
		Size(SizeLong) +
		c.Properties.Size()
}

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CoreInfo) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreInfo) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// The CoreDone event is emitted as a result of a client Sync method.
type CoreDone struct {
	// Passed from [CoreSync.ID].
	ID Int `json:"id"`
	// Passed from [CoreSync.Sequence].
	Sequence Int `json:"seq"`
}

// Size satisfies [KnownSize] with a constant value.
func (c *CoreDone) Size() Word { return SizePrefix + Size(SizeInt) + Size(SizeInt) }

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CoreDone) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreDone) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// The CorePing event is emitted by the server when it wants to check if a client is
// alive or ensure that it has processed the previous events.
type CorePing struct {
	// The object id to ping.
	ID Int `json:"id"`
	// Usually automatically generated.
	// The client should pass this in the Pong method reply.
	Sequence Int `json:"seq"`
}

// Size satisfies [KnownSize] with a constant value.
func (c *CorePing) Size() Word { return SizePrefix + Size(SizeInt) + Size(SizeInt) }

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CorePing) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CorePing) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// The CoreError can be emitted by both the client and the server.
//
// When emitted by the server, the error event is sent out when a fatal
// (non-recoverable) error has occurred. The id argument is the proxy
// object where the error occurred, most often in response to a request
// to that object. The message is a brief description of the error, for
// (debugging) convenience.
//
// When emitted by the client, it indicates an error occurred in an
// object on the client.
type CoreError struct {
	// The id of the resource (proxy if emitted by the client) that is in error.
	ID Int `json:"id"`
	// A seq number from the failing request (if any).
	Sequence Int `json:"seq"`
	// A negative errno style error code.
	Result Int `json:"res"`
	// An error message.
	Message String `json:"message"`
}

func (c *CoreError) Error() string {
	return "received Core::Error on" +
		" id " + strconv.Itoa(int(c.ID)) +
		" seq " + strconv.Itoa(int(c.Sequence)) +
		" res " + strconv.Itoa(int(c.Result)) +
		": " + c.Message
}

// Size satisfies [KnownSize] with a value computed at runtime.
func (c *CoreError) Size() Word {
	return SizePrefix +
		Size(SizeInt) +
		Size(SizeInt) +
		Size(SizeInt) +
		SizeString[Word](c.Message)
}

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CoreError) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreError) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// The CoreBoundProps event is emitted when a local object ID is bound to a global ID.
// It is emitted before the global becomes visible in the registry.
type CoreBoundProps struct {
	// A proxy id.
	ID Int `json:"id"`
	// The global_id as it will appear in the registry.
	GlobalID Int `json:"global_id"`
	// The properties of the global.
	Properties *SPADict `json:"props"`
}

// Size satisfies [KnownSize] with a value computed at runtime.
func (c *CoreBoundProps) Size() Word {
	return SizePrefix +
		Size(SizeInt) +
		Size(SizeInt) +
		c.Properties.Size()
}

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CoreBoundProps) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreBoundProps) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// ErrBadBoundProps is returned when a [CoreBoundProps] event targeting a proxy
// that should never be targeted is received and processed.
var ErrBadBoundProps = errors.New("attempted to store bound props on proxy that should never be targeted")

// noAck is embedded by proxies that are never targeted by [CoreBoundProps].
type noAck struct{}

// setBoundProps should never be called as this proxy should never be targeted by [CoreBoundProps].
func (noAck) setBoundProps(*CoreBoundProps) error { return ErrBadBoundProps }

// An InconsistentIdError describes an inconsistent state where the server claims an impossible
// proxy or global id. This is only generated by the [CoreBoundProps] event.
type InconsistentIdError struct {
	// Whether the inconsistent id is the global resource id.
	Global bool
	// Targeted proxy instance.
	Proxy fmt.Stringer
	// Differing ids.
	ID, ServerID Int
}

func (e *InconsistentIdError) Error() string {
	name := "proxy"
	if e.Global {
		name = "global"
	}

	return name + " id " + strconv.Itoa(int(e.ID)) + " targeting " + e.Proxy.String() +
		" inconsistent with " + strconv.Itoa(int(e.ServerID)) + " claimed by the server"
}

// CoreHello is the first message sent by a client.
type CoreHello struct {
	// The version number of the client, usually PW_VERSION_CORE.
	Version Int `json:"version"`
}

// Size satisfies [KnownSize] with a constant value.
func (c *CoreHello) Size() Word { return SizePrefix + Size(SizeInt) }

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CoreHello) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreHello) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// coreHello queues a [CoreHello] message for the PipeWire server.
// This method should not be called directly, the New function queues this message.
func (ctx *Context) coreHello() error {
	return ctx.writeMessage(
		PW_ID_CORE,
		PW_CORE_METHOD_HELLO,
		&CoreHello{PW_VERSION_CORE},
	)
}

const (
	// CoreSyncSequenceOffset is the offset to [Header.Sequence] to produce [CoreSync.Sequence].
	CoreSyncSequenceOffset = 0x40000000
)

// The CoreSync message will result in a Done event from the server.
// When the Done event is received, the client can be sure that all
// operations before the Sync method have been completed.
type CoreSync struct {
	// The id will be returned in the Done event.
	ID Int `json:"id"`
	// Usually generated automatically and will be returned in the Done event.
	Sequence Int `json:"seq"`
}

// Size satisfies [KnownSize] with a constant value.
func (c *CoreSync) Size() Word { return SizePrefix + Size(SizeInt) + Size(SizeInt) }

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CoreSync) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreSync) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// coreSync queues a [CoreSync] message for the PipeWire server.
// This is not safe to use directly, callers should use Sync instead.
func (ctx *Context) coreSync(id Int) error {
	return ctx.writeMessage(
		PW_ID_CORE,
		PW_CORE_METHOD_SYNC,
		&CoreSync{id, CoreSyncSequenceOffset + Int(ctx.sequence)},
	)
}

// ErrNotDone is returned if [Core.Sync] returns from its [Context.Roundtrip] without
// receiving a [CoreDone] event targeting the [CoreSync] event it delivered.
var ErrNotDone = errors.New("did not receive a Core::Done event targeting previously delivered Core::Sync")

const (
	// syncTimeout is the maximum duration [Core.Sync] is allowed to take before
	// receiving [CoreDone] or failing.
	syncTimeout = 5 * time.Second
)

// Sync queues a [CoreSync] message for the PipeWire server and initiates a Roundtrip.
func (core *Core) Sync() error {
	core.done = false
	if err := core.ctx.coreSync(roundtripSyncID); err != nil {
		return err
	}

	deadline := time.Now().Add(syncTimeout)
	for !core.done {
		if time.Now().After(deadline) {
			return ErrNotDone
		}

		if err := core.ctx.Roundtrip(); err != nil {
			return err
		}
	}
	return nil
}

// The CorePong message is sent from the client to the server when the server emits the Ping event.
type CorePong struct {
	// Copied from [CorePing.ID].
	ID Int `json:"id"`
	// Copied from [CorePing.Sequence]
	Sequence Int `json:"seq"`
}

// Size satisfies [KnownSize] with a constant value.
func (c *CorePong) Size() Word { return SizePrefix + Size(SizeInt) + Size(SizeInt) }

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CorePong) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CorePong) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// CoreGetRegistry is sent when a client requests to bind to the
// registry object and list the available objects on the server.
//
// Like with all bindings, first the client allocates a new proxy
// id and puts this as the new_id field. Methods and Events can
// then be sent and received on the new_id (in the message Id field).
type CoreGetRegistry struct {
	// The version of the registry interface used on the client,
	// usually PW_VERSION_REGISTRY.
	Version Int `json:"version"`
	// The id of the new proxy with the registry interface,
	// ends up as [Header.ID] in future messages.
	NewID Int `json:"new_id"`
}

// Size satisfies [KnownSize] with a constant value.
func (c *CoreGetRegistry) Size() Word { return SizePrefix + Size(SizeInt) + Size(SizeInt) }

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CoreGetRegistry) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreGetRegistry) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// GetRegistry queues a [CoreGetRegistry] message for the PipeWire server
// and returns the address of the newly allocated [Registry].
func (ctx *Context) GetRegistry() (*Registry, error) {
	registry := Registry{Objects: make(map[Int]RegistryGlobal), ctx: ctx}
	newId := ctx.newProxyId(&registry, false)
	registry.ID = newId
	return &registry, ctx.writeMessage(
		PW_ID_CORE,
		PW_CORE_METHOD_GET_REGISTRY,
		&CoreGetRegistry{PW_VERSION_REGISTRY, newId},
	)
}

// A RegistryGlobal event is emitted to notify a client about a new global object.
type RegistryGlobal struct {
	// The global id.
	ID Int `json:"id"`
	// Permission bits.
	Permissions Int `json:"permissions"`
	// The type of object.
	Type String `json:"type"`
	// The server version of the object.
	Version Int `json:"version"`
	// Extra global properties.
	Properties *SPADict `json:"props"`
}

// Size satisfies [KnownSize] with a value computed at runtime.
func (c *RegistryGlobal) Size() Word {
	return SizePrefix +
		Size(SizeInt) +
		Size(SizeInt) +
		SizeString[Word](c.Type) +
		Size(SizeInt) +
		c.Properties.Size()
}

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *RegistryGlobal) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *RegistryGlobal) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// RegistryBind is sent when the client requests to bind to the
// global object with id and use the client proxy with new_id as
// the proxy. After this call, methods can be sent to the remote
// global object and events can be received.
type RegistryBind struct {
	// The [RegistryGlobal.ID] to bind to.
	ID Int `json:"id"`
	// the [RegistryGlobal.Type] of the global id.
	Type String `json:"type"`
	// The client version of the interface for type.
	Version Int `json:"version"`
	// The client proxy id for the global object.
	NewID Int `json:"new_id"`
}

// Size satisfies [KnownSize] with a value computed at runtime.
func (c *RegistryBind) Size() Word {
	return SizePrefix +
		Size(SizeInt) +
		SizeString[Word](c.Type) +
		Size(SizeInt) +
		Size(SizeInt)
}

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *RegistryBind) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *RegistryBind) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// bind queues a [RegistryBind] message for the PipeWire server
// and returns the newly allocated proxy id.
func (registry *Registry) bind(proxy eventProxy, id, version Int) (Int, error) {
	bind := RegistryBind{
		ID:      id,
		Type:    proxy.String(),
		Version: version,
		NewID:   registry.ctx.newProxyId(proxy, true),
	}
	return bind.NewID, registry.ctx.writeMessage(
		registry.ID,
		PW_REGISTRY_METHOD_BIND,
		&bind,
	)
}

// An UnsupportedObjectTypeError is the name of a type not known by the server [Registry].
type UnsupportedObjectTypeError string

func (e UnsupportedObjectTypeError) Error() string { return "unsupported object type " + string(e) }

// Core holds state of [PW_TYPE_INTERFACE_Core].
type Core struct {
	// Additional information from the server, populated or updated during [Context.Roundtrip].
	Info *CoreInfo `json:"info"`

	// Whether a [CoreDone] event was received during Sync.
	done bool

	ctx *Context
	noAck
}

// ErrUnexpectedDone is a [CoreDone] event with unexpected values.
var ErrUnexpectedDone = errors.New("multiple Core::Done events targeting Core::Sync")

// An UnknownBoundIdError describes the server claiming to have bound a proxy id that was never allocated.
type UnknownBoundIdError[E any] struct {
	// Offending id decoded from Data.
	Id Int
	// Event received from the server.
	Event E
}

func (e *UnknownBoundIdError[E]) Error() string {
	return "unknown bound proxy id " + strconv.Itoa(int(e.Id))
}

func (core *Core) consume(opcode byte, files []int, unmarshal func(v any)) error {
	closeReceivedFiles(files...)
	switch opcode {
	case PW_CORE_EVENT_INFO:
		unmarshal(&core.Info)
		return nil

	case PW_CORE_EVENT_DONE:
		var done CoreDone
		unmarshal(&done)
		if done.ID == roundtripSyncID && done.Sequence == CoreSyncSequenceOffset+core.ctx.sequence-1 {
			if core.done {
				return ErrUnexpectedDone
			}
			core.done = true
		}

		// silently ignore non-matching events because the server sends out
		// an event with id -1 seq 0 that does not appear to correspond to
		// anything, and this behaviour is never mentioned in documentation
		return nil

	case PW_CORE_EVENT_ERROR:
		var coreError CoreError
		unmarshal(&coreError)
		return &coreError

	case PW_CORE_EVENT_BOUND_PROPS:
		var boundProps CoreBoundProps
		unmarshal(&boundProps)

		delete(core.ctx.pendingIds, boundProps.ID)
		proxy, ok := core.ctx.proxy[boundProps.ID]
		if !ok {
			return &UnknownBoundIdError[*CoreBoundProps]{Id: boundProps.ID, Event: &boundProps}
		}
		return proxy.setBoundProps(&boundProps)

	default:
		return &UnsupportedOpcodeError{opcode, core.String()}
	}
}

func (core *Core) String() string { return PW_TYPE_INTERFACE_Core }

// Registry holds state of [PW_TYPE_INTERFACE_Registry].
type Registry struct {
	// Proxy id as tracked by [Context].
	ID Int `json:"proxy_id"`

	// Global objects received via the [RegistryGlobal] event.
	//
	// This requires more processing before it can be used, but is not implemented
	// as it is not used by Hakurei.
	Objects map[Int]RegistryGlobal `json:"objects"`

	ctx *Context
	noAck
}

// A GlobalIDCollisionError describes a [RegistryGlobal] event stepping on a previous instance of itself.
type GlobalIDCollisionError struct {
	// The colliding id.
	ID Int
	// Involved events.
	Previous, Current *RegistryGlobal
}

func (e *GlobalIDCollisionError) Error() string {
	return "new Registry::Global event for " + e.Current.Type +
		" stepping on previous id " + strconv.Itoa(int(e.ID)) + " for " + e.Previous.Type
}

func (registry *Registry) consume(opcode byte, files []int, unmarshal func(v any)) error {
	closeReceivedFiles(files...)
	switch opcode {
	case PW_REGISTRY_EVENT_GLOBAL:
		var global RegistryGlobal
		unmarshal(&global)
		if object, ok := registry.Objects[global.ID]; ok {
			// this should never happen so is non-recoverable if it does
			panic(&GlobalIDCollisionError{global.ID, &object, &global})
		}
		registry.Objects[global.ID] = global
		return nil

	default:
		return &UnsupportedOpcodeError{opcode, registry.String()}
	}
}

func (registry *Registry) String() string { return PW_TYPE_INTERFACE_Registry }
