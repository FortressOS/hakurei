package instance

import "hakurei.app/cmd/hakurei/internal/app/internal/setuid"

// ShimMain is the main function of the shim process and runs as the unconstrained target user.
func ShimMain() { setuid.ShimMain() }
