package hst

import (
	"fmt"
	"strconv"
)

const (
	// UserOffset is the offset for UID and GID ranges for each user.
	UserOffset = 100000
	// RangeSize is the size of each UID and GID range.
	RangeSize = UserOffset / 10

	// IdentityStart is the first [Config.Identity] value. This is enforced in cmd/hsu.
	IdentityStart = 0
	// IdentityEnd is the last [Config.Identity] value. This is enforced in cmd/hsu.
	IdentityEnd = AppEnd - AppStart

	// AppStart is the first app user UID and GID.
	AppStart = RangeSize * 1
	// AppEnd is the last app user UID and GID.
	AppEnd = AppStart + RangeSize - 1

	/* these are for Rosa OS: use the ranges below to determine whether a process is isolated */

	// IsolatedStart is the start of UID and GID for fully isolated sandboxed processes.
	IsolatedStart = RangeSize * 9
	// IsolatedEnd is the end of UID and GID for fully isolated sandboxed processes.
	IsolatedEnd = IsolatedStart + RangeSize - 1
)

// A UID represents a kernel uid in the init namespace.
type UID uint32

// String returns the username corresponding to this uid.
//
// Not safe against untrusted input.
func (uid UID) String() string {
	appid := uid % UserOffset
	userid := uid / UserOffset
	if appid >= IsolatedStart && appid <= IsolatedEnd {
		return fmt.Sprintf("u%d_i%d", userid, appid-IsolatedStart)
	} else if appid >= AppStart && appid <= AppEnd {
		return fmt.Sprintf("u%d_a%d", userid, appid-AppStart)
	} else {
		return strconv.Itoa(int(uid))
	}
}

// A GID represents a kernel gid in the init namespace.
type GID uint32

// String returns the group name corresponding to this gid.
//
// Not safe against untrusted input.
func (gid GID) String() string { return UID(gid).String() }

// ToUser returns a [hst.UID] value from userid and appid.
//
// Not safe against untrusted input.
func ToUser[U int | uint32](userid, appid U) U { return userid*UserOffset + AppStart + appid }
