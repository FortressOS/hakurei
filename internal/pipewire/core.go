package pipewire

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

// The FooterClientGeneration indicates to the server what is the last
// registry generation number the client has processed.
//
// The client shall include this footer in the next message it sends,
// after it has processed an incoming message whose footer includes a
// registry generation update.
type FooterClientGeneration struct {
	ClientGeneration Long `json:"client_generation"`
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
	Sequence Int `json:"sequence"`
}

// Size satisfies [KnownSize] with a constant value.
func (c *CoreDone) Size() Word { return SizePrefix + Size(SizeInt) + Size(SizeInt) }

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CoreDone) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreDone) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

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
	Sequence Int `json:"sequence"`
}

// Size satisfies [KnownSize] with a constant value.
func (c *CoreSync) Size() Word { return SizePrefix + Size(SizeInt) + Size(SizeInt) }

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *CoreSync) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreSync) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

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
