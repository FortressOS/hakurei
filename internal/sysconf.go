package internal

//#include <unistd.h>
import "C"

const SC_LOGIN_NAME_MAX = C._SC_LOGIN_NAME_MAX

func Sysconf(name C.int) int { return int(C.sysconf(name)) }
