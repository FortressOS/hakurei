package outcome

import (
	"context"
	"log"
	"time"

	"hakurei.app/hst"
	"hakurei.app/message"
)

// Main runs an app according to [hst.Config] and terminates. Main does not return.
func Main(ctx context.Context, msg message.Msg, config *hst.Config, fd int) {
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
