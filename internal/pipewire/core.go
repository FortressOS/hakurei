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

// CoreHello is the first message sent by a client.
type CoreHello struct {
	// The version number of the client, usually PW_VERSION_CORE.
	Version Int
}

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [MarshalAppend].
func (c *CoreHello) MarshalBinary() ([]byte, error) {
	return MarshalAppend(make([]byte, 0, 24), c)
}

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreHello) UnmarshalBinary(data []byte) error {
	_, err := Unmarshal(data, c)
	return err
}

// CoreGetRegistry is sent when a client requests to bind to the
// registry object and list the available objects on the server.
//
// Like with all bindings, first the client allocates a new proxy
// id and puts this as the new_id field. Methods and Events can
// then be sent and received on the new_id (in the message Id field).
type CoreGetRegistry struct {
	// The version of the registry interface used on the client,
	// usually PW_VERSION_REGISTRY.
	Version Int
	// The id of the new proxy with the registry interface,
	// ends up as [Header.ID] in future messages.
	NewID Int
}

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [MarshalAppend].
func (c *CoreGetRegistry) MarshalBinary() ([]byte, error) {
	return MarshalAppend(make([]byte, 0, 40), c)
}

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *CoreGetRegistry) UnmarshalBinary(data []byte) error {
	_, err := Unmarshal(data, c)
	return err
}
