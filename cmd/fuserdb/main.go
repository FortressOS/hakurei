package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"

	"git.ophivana.moe/security/fortify/internal/fmsg"
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

	type payload struct {
		UserName      string `json:"userName"`
		Uid           int    `json:"uid"`
		Gid           int    `json:"gid"`
		RealName      string `json:"realName"`
		HomeDirectory string `json:"homeDirectory"`
		Shell         string `json:"shell"`
	}

	for _, u := range users {
		fidString := strconv.Itoa(u.fid)
		for aid := 0; aid < 9999; aid++ {
			userName := fmt.Sprintf("u%d_a%d", u.fid, aid)
			uid := 1000000 + u.fid*10000 + aid
			us := strconv.Itoa(uid)
			realName := fmt.Sprintf("Fortify subordinate user %d (%s)", aid, u.name)
			var homeDirectory string
			if *homeDir != varEmpty {
				homeDirectory = path.Join(*homeDir, fidString, strconv.Itoa(aid))
			} else {
				homeDirectory = varEmpty
			}

			fileName := userName + ".user"
			if f, err := os.OpenFile(path.Join(*out, fileName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
				fmsg.Fatalf("cannot create %s: %v", userName, err)
			} else if err = json.NewEncoder(f).Encode(&payload{
				UserName:      userName,
				Uid:           uid,
				Gid:           uid,
				RealName:      realName,
				HomeDirectory: homeDirectory,
				Shell:         *shell,
			}); err != nil {
				fmsg.Fatalf("cannot serialise %s: %v", userName, err)
			} else if err = f.Close(); err != nil {
				fmsg.Printf("cannot close %s: %v", userName, err)
			}
			if err := os.Symlink(fileName, path.Join(*out, us+".user")); err != nil {
				fmsg.Fatalf("cannot link %s: %v", userName, err)
			}
		}
	}

	fmsg.Printf("created %d entries", len(users)*10000)
	fmsg.Exit(0)
}
