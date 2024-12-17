package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"syscall"
)

const (
	compPoison  = "INVALIDINVALIDINVALIDINVALIDINVALID"
	fsuConfFile = "/etc/fsurc"
	envShim     = "FORTIFY_SHIM"
	envAID      = "FORTIFY_APP_ID"
	envGroups   = "FORTIFY_GROUPS"

	PR_SET_NO_NEW_PRIVS = 0x26
)

var (
	Fmain = compPoison
	Fshim = compPoison
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("fsu: ")
	log.SetOutput(os.Stderr)

	if os.Geteuid() != 0 {
		log.Fatal("this program must be owned by uid 0 and have the setuid bit set")
	}

	puid := os.Getuid()
	if puid == 0 {
		log.Fatal("this program must not be started by root")
	}

	var fmain, fshim string
	if p, ok := checkPath(Fmain); !ok {
		log.Fatal("invalid fortify path, this copy of fsu is not compiled correctly")
	} else {
		fmain = p
	}
	if p, ok := checkPath(Fshim); !ok {
		log.Fatal("invalid fshim path, this copy of fsu is not compiled correctly")
	} else {
		fshim = p
	}

	pexe := path.Join("/proc", strconv.Itoa(os.Getppid()), "exe")
	if p, err := os.Readlink(pexe); err != nil {
		log.Fatalf("cannot read parent executable path: %v", err)
	} else if strings.HasSuffix(p, " (deleted)") {
		log.Fatal("fortify executable has been deleted")
	} else if p != fmain {
		log.Fatal("this program must be started by fortify")
	}

	// uid = 1000000 +
	//   fid * 10000 +
	//           aid
	uid := 1000000

	// authenticate before accepting user input
	if fid, ok := parseConfig(fsuConfFile, puid); !ok {
		log.Fatalf("uid %d is not in the fsurc file", puid)
	} else {
		uid += fid * 10000
	}

	// allowed aid range 0 to 9999
	if as, ok := os.LookupEnv(envAID); !ok {
		log.Fatal("FORTIFY_APP_ID not set")
	} else if aid, err := parseUint32Fast(as); err != nil || aid < 0 || aid > 9999 {
		log.Fatal("invalid aid")
	} else {
		uid += aid
	}

	// pass through setup path to shim
	var shimSetupPath string
	if s, ok := os.LookupEnv(envShim); !ok {
		// fortify requests target uid
		// print resolved uid and exit
		fmt.Print(uid)
		os.Exit(0)
	} else if !path.IsAbs(s) {
		log.Fatal("FORTIFY_SHIM is not absolute")
	} else {
		shimSetupPath = s
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

	// careful! users in the allowlist is effectively allowed to drop groups via fsu

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
	if err := syscall.Exec(fshim, []string{"fshim"}, []string{envShim + "=" + shimSetupPath}); err != nil {
		log.Fatalf("cannot start shim: %v", err)
	}

	panic("unreachable")
}

func checkPath(p string) (string, bool) {
	return p, p != compPoison && p != "" && path.IsAbs(p)
}
