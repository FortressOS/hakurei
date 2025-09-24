package sys

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"hakurei.app/container"
	"hakurei.app/hst"
	"hakurei.app/internal"
	"hakurei.app/internal/hlog"
)

// Hsu caches responses from cmd/hsu.
type Hsu struct {
	uidOnce sync.Once
	uidCopy map[int]struct {
		uid int
		err error
	}
	uidMu sync.RWMutex
}

var ErrHsuAccess = errors.New("current user is not in the hsurc file")

func (h *Hsu) Uid(identity int) (int, error) {
	h.uidOnce.Do(func() {
		h.uidCopy = make(map[int]struct {
			uid int
			err error
		})
	})

	{
		h.uidMu.RLock()
		u, ok := h.uidCopy[identity]
		h.uidMu.RUnlock()
		if ok {
			return u.uid, u.err
		}
	}

	h.uidMu.Lock()
	defer h.uidMu.Unlock()

	u := struct {
		uid int
		err error
	}{}
	defer func() { h.uidCopy[identity] = u }()

	u.uid = -1
	hsuPath := internal.MustHsuPath()

	cmd := exec.Command(hsuPath)
	cmd.Path = hsuPath
	cmd.Stderr = os.Stderr // pass through fatal messages
	cmd.Env = []string{"HAKUREI_APP_ID=" + strconv.Itoa(identity)}
	cmd.Dir = container.FHSRoot
	var (
		p         []byte
		exitError *exec.ExitError
	)

	const step = "obtain uid from hsu"
	if p, u.err = cmd.Output(); u.err == nil {
		u.uid, u.err = strconv.Atoi(string(p))
		if u.err != nil {
			u.err = &hst.AppError{Step: step, Err: u.err, Msg: "invalid uid string from hsu"}
		}
	} else if errors.As(u.err, &exitError) && exitError != nil && exitError.ExitCode() == 1 {
		// hsu prints an error message in this case
		u.err = &hst.AppError{Step: step, Err: ErrHsuAccess}
	} else if os.IsNotExist(u.err) {
		u.err = &hst.AppError{Step: step, Err: os.ErrNotExist,
			Msg: fmt.Sprintf("the setuid helper is missing: %s", hsuPath)}
	}
	return u.uid, u.err
}

// MustUid calls [State.Uid] and terminates on error.
func MustUid(s State, identity int) int {
	uid, err := s.Uid(identity)
	if err == nil {
		return uid
	}

	const fallback = "cannot obtain uid from setuid wrapper:"
	if errors.Is(err, ErrHsuAccess) {
		hlog.Verbose("*"+fallback, err)
		os.Exit(1)
		return -0xdeadbeef
	} else if m, ok := container.GetErrorMessage(err); ok {
		log.Fatal(m)
		return -0xdeadbeef
	} else {
		log.Fatalln(fallback, err)
		return -0xdeadbeef
	}
}
