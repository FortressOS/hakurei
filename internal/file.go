package internal

import "syscall"

// IgnoringEINTR makes a function call and repeats it if it returns an
// EINTR error. This appears to be required even though we install all
// signal handlers with SA_RESTART: see #22838, #38033, #38836, #40846.
// Also #20400 and #36644 are issues in which a signal handler is
// installed without setting SA_RESTART. None of these are the common case,
// but there are enough of them that it seems that we can't avoid
// an EINTR loop.
func IgnoringEINTR(fn func() error) error {
	for {
		err := fn()
		if err != syscall.EINTR {
			return err
		}
	}
}
