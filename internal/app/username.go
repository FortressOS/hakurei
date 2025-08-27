package app

import "regexp"

// nameRegex is the default NAME_REGEX value from adduser.
var nameRegex = regexp.MustCompilePOSIX(`^[a-zA-Z][a-zA-Z0-9_-]*\$?$`)

// isValidUsername returns whether the argument is a valid username
func isValidUsername(username string) bool {
	return len(username) < sysconf(_SC_LOGIN_NAME_MAX) &&
		nameRegex.MatchString(username)
}
