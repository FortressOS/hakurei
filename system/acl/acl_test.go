package acl_test

import (
	"errors"
	"os"
	"path"
	"reflect"
	"testing"

	"git.gensokyo.uk/security/hakurei/system/acl"
)

const testFileName = "acl.test"

var (
	uid  = os.Geteuid()
	cred = int32(os.Geteuid())
)

func TestUpdatePerm(t *testing.T) {
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
			t.Fatalf("UpdatePerm: error = %v", err)
		}
		if cur = getfacl(t, testFilePath); len(cur) != 4 {
			t.Fatalf("UpdatePerm: %v", cur)
		}
	})

	t.Run("default clear consistency", func(t *testing.T) {
		if err := acl.Update(testFilePath, uid); err != nil {
			t.Fatalf("UpdatePerm: error = %v", err)
		}
		if val := getfacl(t, testFilePath); !reflect.DeepEqual(val, cur) {
			t.Fatalf("UpdatePerm: %v, want %v", val, cur)
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
				t.Fatalf("UpdatePerm: error = %v", err)
			}
			if v := getfacl(t, testFilePath); !reflect.DeepEqual(v, cur) {
				t.Fatalf("UpdatePerm: %v, want %v", v, cur)
			}
		})

		if err := acl.Update(testFilePath, uid, perms...); err != nil {
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
