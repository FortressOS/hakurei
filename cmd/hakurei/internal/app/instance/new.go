// Package instance exposes cross-package implementation details and provides constructors for builtin implementations.
package instance

import (
	"context"
	"log"
	"syscall"

	"git.gensokyo.uk/security/hakurei/cmd/hakurei/internal/app"
	"git.gensokyo.uk/security/hakurei/cmd/hakurei/internal/app/internal/setuid"
	"git.gensokyo.uk/security/hakurei/internal/sys"
)

const (
	ISetuid = iota
)

func New(whence int, ctx context.Context, os sys.State) (app.App, error) {
	switch whence {
	case ISetuid:
		return setuid.New(ctx, os)
	default:
		return nil, syscall.EINVAL
	}
}

func MustNew(whence int, ctx context.Context, os sys.State) app.App {
	a, err := New(whence, ctx, os)
	if err != nil {
		log.Fatalf("cannot create app: %v", err)
	}
	return a
}
