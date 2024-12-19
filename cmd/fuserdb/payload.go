package main

import (
	"encoding/json"
	"os"
	"path"

	"git.gensokyo.uk/security/fortify/internal/fmsg"
)

type payloadU struct {
	UserName      string   `json:"userName"`
	Uid           int      `json:"uid"`
	Gid           int      `json:"gid"`
	MemberOf      []string `json:"memberOf,omitempty"`
	RealName      string   `json:"realName"`
	HomeDirectory string   `json:"homeDirectory"`
	Shell         string   `json:"shell"`
}

func writeUser(userName string, uid int, us string, realName, homeDirectory, shell string, out string) {
	userFileName := userName + ".user"
	if f, err := os.OpenFile(path.Join(out, userFileName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		fmsg.Fatalf("cannot create %s: %v", userName, err)
	} else if err = json.NewEncoder(f).Encode(&payloadU{
		UserName:      userName,
		Uid:           uid,
		Gid:           uid,
		RealName:      realName,
		HomeDirectory: homeDirectory,
		Shell:         shell,
	}); err != nil {
		fmsg.Fatalf("cannot serialise %s: %v", userName, err)
	} else if err = f.Close(); err != nil {
		fmsg.Printf("cannot close %s: %v", userName, err)
	}
	if err := os.Symlink(userFileName, path.Join(out, us+".user")); err != nil {
		fmsg.Fatalf("cannot link %s: %v", userName, err)
	}
}

type payloadG struct {
	GroupName string   `json:"groupName"`
	Gid       int      `json:"gid"`
	Members   []string `json:"members,omitempty"`
}

func writeGroup(groupName string, gid int, gs string, members []string, out string) {
	groupFileName := groupName + ".group"
	if f, err := os.OpenFile(path.Join(out, groupFileName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		fmsg.Fatalf("cannot create %s: %v", groupName, err)
	} else if err = json.NewEncoder(f).Encode(&payloadG{
		GroupName: groupName,
		Gid:       gid,
		Members:   members,
	}); err != nil {
		fmsg.Fatalf("cannot serialise %s: %v", groupName, err)
	} else if err = f.Close(); err != nil {
		fmsg.Printf("cannot close %s: %v", groupName, err)
	}
	if err := os.Symlink(groupFileName, path.Join(out, gs+".group")); err != nil {
		fmsg.Fatalf("cannot link %s: %v", groupName, err)
	}
}
