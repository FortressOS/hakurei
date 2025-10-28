package validate

import "regexp"

// nameRegex is the default NAME_REGEX value from adduser.
var nameRegex = regexp.MustCompilePOSIX(`^[a-zA-Z][a-zA-Z0-9_-]*\$?$`)

// IsValidUsername returns whether the argument is a valid username.
func IsValidUsername(username string) bool {
	return len(username) < Sysconf(SC_LOGIN_NAME_MAX) &&
		nameRegex.MatchString(username)
}
