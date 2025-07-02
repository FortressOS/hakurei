package acl_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
)

type (
	getFAclInvocation struct {
		cmd *exec.Cmd
		val []*getFAclResp
		pe  []error
	}

	getFAclResp struct {
		typ  fAclType
		cred int32
		val  fAclPerm

		raw []byte
	}

	fAclPerm uintptr
	fAclType uint8
)

const fAclBufSize = 16

const (
	fAclPermRead fAclPerm = 1 << iota
	fAclPermWrite
	fAclPermExecute
)

const (
	fAclTypeUser fAclType = iota
	fAclTypeGroup
	fAclTypeMask
	fAclTypeOther
)

func (c *getFAclInvocation) run(name string) error {
	if c.cmd != nil {
		panic("attempted to run twice")
	}

	c.cmd = exec.Command("getfacl", "--omit-header", "--absolute-names", "--numeric", name)

	scanErr := make(chan error, 1)
	if p, err := c.cmd.StdoutPipe(); err != nil {
		return err
	} else {
		go c.parse(p, scanErr)
	}

	if err := c.cmd.Start(); err != nil {
		return err
	}

	return errors.Join(<-scanErr, c.cmd.Wait())
}

func (c *getFAclInvocation) parse(pipe io.Reader, scanErr chan error) {
	c.val = make([]*getFAclResp, 0, 4+fAclBufSize)

	s := bufio.NewScanner(pipe)
	for s.Scan() {
		fields := bytes.SplitN(s.Bytes(), []byte{':'}, 3)
		if len(fields) != 3 {
			continue
		}

		resp := getFAclResp{}

		switch string(fields[0]) {
		case "user":
			resp.typ = fAclTypeUser
		case "group":
			resp.typ = fAclTypeGroup
		case "mask":
			resp.typ = fAclTypeMask
		case "other":
			resp.typ = fAclTypeOther
		default:
			c.pe = append(c.pe, fmt.Errorf("unknown type %s", string(fields[0])))
			continue
		}

		if len(fields[1]) == 0 {
			resp.cred = -1
		} else {
			if cred, err := strconv.Atoi(string(fields[1])); err != nil {
				c.pe = append(c.pe, err)
				continue
			} else {
				resp.cred = int32(cred)
				if resp.cred < 0 {
					c.pe = append(c.pe, fmt.Errorf("credential %d out of range", resp.cred))
					continue
				}
			}
		}

		if len(fields[2]) != 3 {
			c.pe = append(c.pe, fmt.Errorf("invalid perm length %d", len(fields[2])))
			continue
		} else {
			switch fields[2][0] {
			case 'r':
				resp.val |= fAclPermRead
			case '-':
			default:
				c.pe = append(c.pe, fmt.Errorf("invalid perm %v", fields[2][0]))
				continue
			}
			switch fields[2][1] {
			case 'w':
				resp.val |= fAclPermWrite
			case '-':
			default:
				c.pe = append(c.pe, fmt.Errorf("invalid perm %v", fields[2][1]))
				continue
			}
			switch fields[2][2] {
			case 'x':
				resp.val |= fAclPermExecute
			case '-':
			default:
				c.pe = append(c.pe, fmt.Errorf("invalid perm %v", fields[2][2]))
				continue
			}
		}

		resp.raw = make([]byte, len(s.Bytes()))
		copy(resp.raw, s.Bytes())
		c.val = append(c.val, &resp)
	}
	scanErr <- s.Err()
}

func (r *getFAclResp) String() string {
	if r.raw != nil && len(r.raw) > 0 {
		return string(r.raw)
	}

	return "(user-initialised resp value)"
}

func (r *getFAclResp) equals(typ fAclType, cred int32, val fAclPerm) bool {
	return r.typ == typ && r.cred == cred && r.val == val
}
