package main

// minimise imports to avoid inadvertently calling init or global variable functions

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
)

const (
	// envIdentity is the name of the environment variable holding a
	// single byte representing the shim setup pipe file descriptor.
	envShim = "HAKUREI_SHIM"
	// envGroups holds a ' ' separated list of string representations of
	// supplementary group gid. Membership requirements are enforced.
	envGroups = "HAKUREI_GROUPS"
)

// hakureiPath is the absolute path to Hakurei.
//
// This is set by the linker.
var hakureiPath string

func main() {
	const PR_SET_NO_NEW_PRIVS = 0x26
	runtime.LockOSThread()

	log.SetFlags(0)
	log.SetPrefix("hsu: ")
	log.SetOutput(os.Stderr)

	if os.Geteuid() != 0 {
		log.Fatal("this program must be owned by uid 0 and have the setuid bit set")
	}
	if os.Getegid() != os.Getgid() {
		log.Fatal("this program must not have the setgid bit set")
	}

	puid := os.Getuid()
	if puid == 0 {
		log.Fatal("this program must not be started by root")
	}

	if !path.IsAbs(hakureiPath) {
		log.Fatal("this program is compiled incorrectly")
		return
	}

	var toolPath string
	pexe := path.Join("/proc", strconv.Itoa(os.Getppid()), "exe")
	if p, err := os.Readlink(pexe); err != nil {
		log.Fatalf("cannot read parent executable path: %v", err)
	} else if strings.HasSuffix(p, " (deleted)") {
		log.Fatal("hakurei executable has been deleted")
	} else if p != hakureiPath {
		log.Fatal("this program must be started by hakurei")
	} else {
		toolPath = p
	}

	// refuse to run if hsurc is not protected correctly
	if s, err := os.Stat(hsuConfPath); err != nil {
		log.Fatal(err)
	} else if s.Mode().Perm() != 0400 {
		log.Fatal("bad hsurc perm")
	} else if st := s.Sys().(*syscall.Stat_t); st.Uid != 0 || st.Gid != 0 {
		log.Fatal("hsurc must be owned by uid 0")
	}

	// authenticate before accepting user input
	userid := mustParseConfig(puid)

	// pass through setup fd to shim
	var shimSetupFd string
	if s, ok := os.LookupEnv(envShim); !ok {
		// hakurei requests hsurc user id
		fmt.Print(userid)
		os.Exit(0)
	} else if len(s) != 1 || s[0] > '9' || s[0] < '3' {
		log.Fatal("HAKUREI_SHIM holds an invalid value")
	} else {
		shimSetupFd = s
	}

	// start is going ahead at this point
	identity := mustReadIdentity()

	const (
		// first possible uid outcome
		uidStart = 10000
		// last possible uid outcome
		uidEnd = 999919999
	)

	// cast to int for use with library functions
	uid := int(toUser(userid, identity))

	// final bounds check to catch any bugs
	if uid < uidStart || uid >= uidEnd {
		panic("uid out of bounds")
	}

	// supplementary groups
	var suppGroups, suppCurrent []int

	if gs, ok := os.LookupEnv(envGroups); ok {
		if cur, err := os.Getgroups(); err != nil {
			log.Fatalf("cannot get groups: %v", err)
		} else {
			suppCurrent = cur
		}

		// parse space-separated list of group ids
		gss := bytes.Split([]byte(gs), []byte{' '})
		suppGroups = make([]int, len(gss)+1)
		for i, s := range gss {
			if gid, err := strconv.Atoi(string(s)); err != nil {
				log.Fatalf("cannot parse %q: %v", string(s), err)
			} else if gid > 0 && gid != uid && gid != os.Getgid() && slices.Contains(suppCurrent, gid) {
				suppGroups[i] = gid
			} else {
				log.Fatalf("invalid gid %d", gid)
			}
		}
		suppGroups[len(suppGroups)-1] = uid
	} else {
		suppGroups = []int{uid}
	}

	// careful! users in the allowlist is effectively allowed to drop groups via hsu

	if err := syscall.Setresgid(uid, uid, uid); err != nil {
		log.Fatalf("cannot set gid: %v", err)
	}
	if err := syscall.Setgroups(suppGroups); err != nil {
		log.Fatalf("cannot set supplementary groups: %v", err)
	}
	if err := syscall.Setresuid(uid, uid, uid); err != nil {
		log.Fatalf("cannot set uid: %v", err)
	}
	if _, _, errno := syscall.AllThreadsSyscall(syscall.SYS_PRCTL, PR_SET_NO_NEW_PRIVS, 1, 0); errno != 0 {
		log.Fatalf("cannot set no_new_privs flag: %s", errno.Error())
	}
	if err := syscall.Exec(toolPath, []string{"hakurei", "shim"}, []string{envShim + "=" + shimSetupFd}); err != nil {
		log.Fatalf("cannot start shim: %v", err)
	}

	panic("unreachable")
}
