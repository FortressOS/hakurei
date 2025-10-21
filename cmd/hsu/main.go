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
	hsuConfFile = "/etc/hsurc"
	envShim     = "HAKUREI_SHIM"
	envIdentity = "HAKUREI_IDENTITY"
	envGroups   = "HAKUREI_GROUPS"

	PR_SET_NO_NEW_PRIVS = 0x26

	identityMin = 0
	identityMax = 9999
)

// hakureiPath is the absolute path to Hakurei.
//
// This is set by the linker.
var hakureiPath string

func main() {
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

	// uid = 1000000 +
	//    id * 10000 +
	//      identity
	uid := 1000000

	// refuse to run if hsurc is not protected correctly
	if s, err := os.Stat(hsuConfFile); err != nil {
		log.Fatal(err)
	} else if s.Mode().Perm() != 0400 {
		log.Fatal("bad hsurc perm")
	} else if st := s.Sys().(*syscall.Stat_t); st.Uid != 0 || st.Gid != 0 {
		log.Fatal("hsurc must be owned by uid 0")
	}

	// authenticate before accepting user input
	var id int
	if f, err := os.Open(hsuConfFile); err != nil {
		log.Fatal(err)
	} else if v, ok := mustParseConfig(f, puid); !ok {
		log.Fatalf("uid %d is not in the hsurc file", puid)
	} else {
		id = v
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}

		uid += id * 10000
	}

	// pass through setup fd to shim
	var shimSetupFd string
	if s, ok := os.LookupEnv(envShim); !ok {
		// hakurei requests hsurc user id
		fmt.Print(id)
		os.Exit(0)
	} else if len(s) != 1 || s[0] > '9' || s[0] < '3' {
		log.Fatal("HAKUREI_SHIM holds an invalid value")
	} else {
		shimSetupFd = s
	}

	// allowed identity range 0 to 9999
	if as, ok := os.LookupEnv(envIdentity); !ok {
		log.Fatal("HAKUREI_IDENTITY not set")
	} else if identity, err := parseUint32Fast(as); err != nil || identity < identityMin || identity > identityMax {
		log.Fatal("invalid identity")
	} else {
		uid += identity
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

	// final bounds check to catch any bugs
	if uid < 1000000 || uid >= 2000000 {
		panic("uid out of bounds")
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
