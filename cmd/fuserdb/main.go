package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

func main() {
	fmsg.SetPrefix("fuserdb")

	const varEmpty = "/var/empty"

	out := flag.String("o", "userdb", "output directory")
	homeDir := flag.String("d", varEmpty, "parent of home directories")
	shell := flag.String("s", "/sbin/nologin", "absolute path to subordinate user shell")
	flag.Parse()

	type user struct {
		name string
		fid  int
	}

	users := make([]user, len(flag.Args()))
	for i, s := range flag.Args() {
		f := bytes.SplitN([]byte(s), []byte{':'}, 2)
		if len(f) != 2 {
			fmsg.Fatalf("invalid entry at index %d", i)
		}
		users[i].name = string(f[0])
		if fid, err := strconv.Atoi(string(f[1])); err != nil {
			fmsg.Fatal(err.Error())
		} else {
			users[i].fid = fid
		}
	}

	if err := os.MkdirAll(*out, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		fmsg.Fatalf("cannot create output: %v", err)
	}

	for _, u := range users {
		fidString := strconv.Itoa(u.fid)
		for aid := 0; aid < 10000; aid++ {
			userName := fmt.Sprintf("u%d_a%d", u.fid, aid)
			uid := 1000000 + u.fid*10000 + aid
			us := strconv.Itoa(uid)
			realName := fmt.Sprintf("Fortify subordinate user %d (%s)", aid, u.name)
			var homeDirectory string
			if *homeDir != varEmpty {
				homeDirectory = path.Join(*homeDir, "u"+fidString, "a"+strconv.Itoa(aid))
			} else {
				homeDirectory = varEmpty
			}

			writeUser(userName, uid, us, realName, homeDirectory, *shell, *out)
			writeGroup(userName, uid, us, nil, *out)
		}
	}

	fmsg.Printf("created %d entries", len(users)*2*10000)
	fmsg.Exit(0)
}
