package dbus

import (
	"context"
	"io"
)

// NewDirect returns a new instance of [Proxy] with its sandbox disabled.
func NewDirect(ctx context.Context, final *Final, output io.Writer) *Proxy {
	p := New(ctx, final, output)
	p.useSandbox = false
	return p
}
