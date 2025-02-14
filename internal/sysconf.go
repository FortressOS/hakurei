package internal

//#include <unistd.h>
import "C"

func Sysconf_SC_LOGIN_NAME_MAX() int { return int(C.sysconf(C._SC_LOGIN_NAME_MAX)) }
