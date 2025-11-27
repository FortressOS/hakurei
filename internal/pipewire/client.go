package pipewire

/* pipewire/client.h */

const (
	PW_TYPE_INTERFACE_Client = PW_TYPE_INFO_INTERFACE_BASE + "Client"
	PW_CLIENT_PERM_MASK      = PW_PERM_RWXM
	PW_VERSION_CLIENT        = 3

	PW_ID_CLIENT = 1
)

const (
	PW_CLIENT_CHANGE_MASK_PROPS = 1 << iota

	PW_CLIENT_CHANGE_MASK_ALL = 1<<iota - 1
)

const (
	PW_CLIENT_EVENT_INFO = iota
	PW_CLIENT_EVENT_PERMISSIONS
	PW_CLIENT_EVENT_NUM

	PW_VERSION_CLIENT_EVENTS = 0
)

const (
	PW_CLIENT_METHOD_ADD_LISTENER = iota
	PW_CLIENT_METHOD_ERROR
	PW_CLIENT_METHOD_UPDATE_PROPERTIES
	PW_CLIENT_METHOD_GET_PERMISSIONS
	PW_CLIENT_METHOD_UPDATE_PERMISSIONS
	PW_CLIENT_METHOD_NUM

	PW_VERSION_CLIENT_METHODS = 0
)

// The ClientInfo event provides client information updates.
// This is emitted when binding to a client or when the client info is updated later.
type ClientInfo struct {
	// The global id of the client.
	ID Int `json:"id"`
	// The changes emitted by this event.
	ChangeMask Long `json:"change_mask"`
	// Properties of this object, valid when change_mask has PW_CLIENT_CHANGE_MASK_PROPS.
	Properties *SPADict `json:"props"`
}

// Size satisfies [KnownSize] with a value computed at runtime.
func (c *ClientInfo) Size() Word {
	return SizePrefix +
		Size(SizeInt) +
		Size(SizeLong) +
		c.Properties.Size()
}

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *ClientInfo) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *ClientInfo) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }

// ClientUpdateProperties is used to update the properties of a client.
type ClientUpdateProperties struct {
	// Properties to update on the client.
	Properties *SPADict `json:"props"`
}

// Size satisfies [KnownSize] with a value computed at runtime.
func (c *ClientUpdateProperties) Size() Word { return SizePrefix + c.Properties.Size() }

// MarshalBinary satisfies [encoding.BinaryMarshaler] via [Marshal].
func (c *ClientUpdateProperties) MarshalBinary() ([]byte, error) { return Marshal(c) }

// UnmarshalBinary satisfies [encoding.BinaryUnmarshaler] via [Unmarshal].
func (c *ClientUpdateProperties) UnmarshalBinary(data []byte) error { return Unmarshal(data, c) }
