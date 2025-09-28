package dbus

import (
	"context"
	"io"

	"hakurei.app/container"
)

// NewDirect returns a new instance of [Proxy] with its sandbox disabled.
func NewDirect(ctx context.Context, msg container.Msg, final *Final, output io.Writer) *Proxy {
	p := New(ctx, msg, final, output)
	p.useSandbox = false
	return p
}
