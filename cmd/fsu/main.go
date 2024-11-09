package main

import (
	"bufio"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
)

const (
	compPoison  = "INVALIDINVALIDINVALIDINVALIDINVALID"
	fsuConfFile = "/etc/fsurc"
	envShim     = "FORTIFY_SHIM"
	envAID      = "FORTIFY_APP_ID"

	PR_SET_NO_NEW_PRIVS = 0x26
)

var Fmain = compPoison

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

	var fmain string
	if p, ok := checkPath(Fmain); !ok {
		log.Fatal("invalid fortify path, this copy of fsu is not compiled correctly")
	} else {
		fmain = p
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

	// pass through setup path to shim
	var shimSetupPath string
	if s, ok := os.LookupEnv(envShim); !ok {
		log.Fatal("FORTIFY_SHIM not set")
	} else if !path.IsAbs(s) {
		log.Fatal("FORTIFY_SHIM is not absolute")
	} else {
		shimSetupPath = s
	}

	// allowed aid range 0 to 9999
	if as, ok := os.LookupEnv(envAID); !ok {
		log.Fatal("FORTIFY_APP_ID not set")
	} else if aid, err := strconv.Atoi(as); err != nil || aid < 0 || aid > 9999 {
		log.Fatal("invalid aid")
	} else {
		uid += aid
	}

	if err := syscall.Setresgid(uid, uid, uid); err != nil {
		log.Fatalf("cannot set gid: %v", err)
	}
	if err := syscall.Setresuid(uid, uid, uid); err != nil {
		log.Fatalf("cannot set uid: %v", err)
	}
	if _, _, errno := syscall.AllThreadsSyscall(syscall.SYS_PRCTL, PR_SET_NO_NEW_PRIVS, 1, 0); errno != 0 {
		log.Fatalf("cannot set no_new_privs flag: %s", errno.Error())
	}
	if err := syscall.Exec(fmain, []string{"fortify", "shim"}, []string{envShim + "=" + shimSetupPath}); err != nil {
		log.Fatalf("cannot start shim: %v", err)
	}

	panic("unreachable")
}

func parseConfig(p string, puid int) (fid int, ok bool) {
	// refuse to run if fsurc is not protected correctly
	if s, err := os.Stat(p); err != nil {
		log.Fatal(err)
	} else if s.Mode().Perm() != 0400 {
		log.Fatal("bad fsurc perm")
	} else if st := s.Sys().(*syscall.Stat_t); st.Uid != 0 || st.Gid != 0 {
		log.Fatal("fsurc must be owned by uid 0")
	}

	if r, err := os.Open(p); err != nil {
		log.Fatal(err)
		return -1, false
	} else {
		s := bufio.NewScanner(r)
		var line int
		for s.Scan() {
			line++

			// <puid> <fid>
			lf := strings.SplitN(s.Text(), " ", 2)
			if len(lf) != 2 {
				log.Fatalf("invalid entry on line %d", line)
			}

			var puid0 int
			if puid0, err = strconv.Atoi(lf[0]); err != nil || puid0 < 1 {
				log.Fatalf("invalid parent uid on line %d", line)
			}

			ok = puid0 == puid
			if ok {
				// allowed fid range 0 to 99
				if fid, err = strconv.Atoi(lf[1]); err != nil || fid < 0 || fid > 99 {
					log.Fatalf("invalid fortify uid on line %d", line)
				}
				return
			}
		}
		if err = s.Err(); err != nil {
			log.Fatalf("cannot read fsurc: %v", err)
		}
		return -1, false
	}
}

func checkPath(p string) (string, bool) {
	return p, p != compPoison && p != "" && path.IsAbs(p)
}
