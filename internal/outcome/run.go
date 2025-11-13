package outcome

import (
	"context"
	"log"
	"time"
	_ "unsafe" // for go:linkname

	"hakurei.app/hst"
	"hakurei.app/message"
)

// IsPollDescriptor reports whether fd is the descriptor being used by the poller.
//
// Made available here to determine and reject impossible fd.
//
//go:linkname IsPollDescriptor internal/poll.IsPollDescriptor
func IsPollDescriptor(fd uintptr) bool

// Main runs an app according to [hst.Config] and terminates. Main does not return.
func Main(ctx context.Context, msg message.Msg, config *hst.Config, fd int) {
	// avoids runtime internals or standard streams
	if fd >= 0 {
		if IsPollDescriptor(uintptr(fd)) || fd < 3 {
			log.Fatalf("invalid identifier fd %d", fd)
		}
	}

	var id hst.ID
	if err := hst.NewInstanceID(&id); err != nil {
		log.Fatal(err.Error())
	}

	k := outcome{syscallDispatcher: direct{msg}}

	finaliseTime := time.Now()
	if err := k.finalise(ctx, msg, &id, config); err != nil {
		printMessageError(msg.GetLogger().Fatalln, "cannot seal app:", err)
		panic("unreachable")
	}
	msg.Verbosef("finalise took %.2f ms", float64(time.Since(finaliseTime).Nanoseconds())/1e6)

	k.main(msg, fd)
	panic("unreachable")
}
