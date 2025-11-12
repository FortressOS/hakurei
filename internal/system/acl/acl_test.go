package acl_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strconv"
	"testing"

	"hakurei.app/internal/system/acl"
)

const testFileName = "acl.test"

var (
	uid  = os.Geteuid()
	cred = int32(os.Geteuid())
)

func TestUpdate(t *testing.T) {
	if os.Getenv("GO_TEST_SKIP_ACL") == "1" {
		t.Log("acl test skipped")
		t.SkipNow()
	}

	testFilePath := path.Join(t.TempDir(), testFileName)

	if f, err := os.Create(testFilePath); err != nil {
		t.Fatalf("Create: error = %v", err)
	} else {
		if err = f.Close(); err != nil {
			t.Fatalf("Close: error = %v", err)
		}
	}
	defer func() {
		if err := os.Remove(testFilePath); err != nil {
			t.Fatalf("Remove: error = %v", err)
		}
	}()

	cur := getfacl(t, testFilePath)

	t.Run("default entry count", func(t *testing.T) {
		if len(cur) != 3 {
			t.Fatalf("unexpected test file acl length %d", len(cur))
		}
	})

	t.Run("default clear mask", func(t *testing.T) {
		if err := acl.Update(testFilePath, uid); err != nil {
			t.Fatalf("Update: error = %v", err)
		}
		if cur = getfacl(t, testFilePath); len(cur) != 4 {
			t.Fatalf("Update: %v", cur)
		}
	})

	t.Run("default clear consistency", func(t *testing.T) {
		if err := acl.Update(testFilePath, uid); err != nil {
			t.Fatalf("Update: error = %v", err)
		}
		if val := getfacl(t, testFilePath); !reflect.DeepEqual(val, cur) {
			t.Fatalf("Update: %v, want %v", val, cur)
		}
	})

	testUpdate(t, testFilePath, "r--", cur, fAclPermRead, acl.Read)
	testUpdate(t, testFilePath, "-w-", cur, fAclPermWrite, acl.Write)
	testUpdate(t, testFilePath, "--x", cur, fAclPermExecute, acl.Execute)
	testUpdate(t, testFilePath, "-wx", cur, fAclPermWrite|fAclPermExecute, acl.Write, acl.Execute)
	testUpdate(t, testFilePath, "r-x", cur, fAclPermRead|fAclPermExecute, acl.Read, acl.Execute)
	testUpdate(t, testFilePath, "rw-", cur, fAclPermRead|fAclPermWrite, acl.Read, acl.Write)
	testUpdate(t, testFilePath, "rwx", cur, fAclPermRead|fAclPermWrite|fAclPermExecute, acl.Read, acl.Write, acl.Execute)
}

func testUpdate(t *testing.T, testFilePath, name string, cur []*getFAclResp, val fAclPerm, perms ...acl.Perm) {
	t.Run(name, func(t *testing.T) {
		t.Cleanup(func() {
			if err := acl.Update(testFilePath, uid); err != nil {
				t.Fatalf("Update: error = %v", err)
			}
			if v := getfacl(t, testFilePath); !reflect.DeepEqual(v, cur) {
				t.Fatalf("Update: %v, want %v", v, cur)
			}
		})

		if err := acl.Update(testFilePath, uid, perms...); err != nil {
			t.Fatalf("Update: error = %v", err)
		}
		r := respByCred(getfacl(t, testFilePath), fAclTypeUser, cred)
		if r == nil {
			t.Fatalf("Update did not add an ACL entry")
		}
		if !r.equals(fAclTypeUser, cred, val) {
			t.Fatalf("Update(%s) = %s", name, r)
		}
	})
}

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

func getfacl(t *testing.T, name string) []*getFAclResp {
	c := new(getFAclInvocation)
	if err := c.run(name); err != nil {
		t.Fatalf("getfacl: error = %v", err)
	}
	if len(c.pe) != 0 {
		t.Errorf("errors encountered parsing getfacl output\n%s", errors.Join(c.pe...).Error())
	}
	return c.val
}

func respByCred(v []*getFAclResp, typ fAclType, cred int32) *getFAclResp {
	j := -1
	for i, r := range v {
		if r.typ == typ && r.cred == cred {
			if j != -1 {
				panic("invalid acl")
			}
			j = i
		}
	}
	if j == -1 {
		return nil
	}
	return v[j]
}
