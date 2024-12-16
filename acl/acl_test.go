package acl_test

import (
	"errors"
	"os"
	"path"
	"reflect"
	"testing"

	"git.ophivana.moe/security/fortify/acl"
)

const testFileName = "acl.test"

var (
	uid  = os.Geteuid()
	cred = int32(os.Geteuid())

	testFilePath = path.Join(os.TempDir(), testFileName)
)

func TestUpdatePerm(t *testing.T) {

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
		if err := acl.UpdatePerm(testFilePath, uid); err != nil {
			t.Fatalf("UpdatePerm: error = %v", err)
		}
		if cur = getfacl(t, testFilePath); len(cur) != 4 {
			t.Fatalf("UpdatePerm: %v", cur)
		}
	})

	t.Run("default clear consistency", func(t *testing.T) {
		if err := acl.UpdatePerm(testFilePath, uid); err != nil {
			t.Fatalf("UpdatePerm: error = %v", err)
		}
		if val := getfacl(t, testFilePath); !reflect.DeepEqual(val, cur) {
			t.Fatalf("UpdatePerm: %v, want %v", val, cur)
		}
	})

	testUpdate(t, "r--", cur, fAclPermRead, acl.Read)
	testUpdate(t, "-w-", cur, fAclPermWrite, acl.Write)
	testUpdate(t, "--x", cur, fAclPermExecute, acl.Execute)
	testUpdate(t, "-wx", cur, fAclPermWrite|fAclPermExecute, acl.Write, acl.Execute)
	testUpdate(t, "r-x", cur, fAclPermRead|fAclPermExecute, acl.Read, acl.Execute)
	testUpdate(t, "rw-", cur, fAclPermRead|fAclPermWrite, acl.Read, acl.Write)
	testUpdate(t, "rwx", cur, fAclPermRead|fAclPermWrite|fAclPermExecute, acl.Read, acl.Write, acl.Execute)
}

func testUpdate(t *testing.T, name string, cur []*getFAclResp, val fAclPerm, perms ...acl.Perm) {
	t.Run(name, func(t *testing.T) {
		t.Cleanup(func() {
			if err := acl.UpdatePerm(testFilePath, uid); err != nil {
				t.Fatalf("UpdatePerm: error = %v", err)
			}
			if v := getfacl(t, testFilePath); !reflect.DeepEqual(v, cur) {
				t.Fatalf("UpdatePerm: %v, want %v", v, cur)
			}
		})

		if err := acl.UpdatePerm(testFilePath, uid, perms...); err != nil {
			t.Fatalf("UpdatePerm: error = %v", err)
		}
		r := respByCred(getfacl(t, testFilePath), fAclTypeUser, cred)
		if r == nil {
			t.Fatalf("UpdatePerm did not add an ACL entry")
		}
		if !r.equals(fAclTypeUser, cred, val) {
			t.Fatalf("UpdatePerm(%s) = %s", name, r)
		}
	})
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
