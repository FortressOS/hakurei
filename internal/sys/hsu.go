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
	idOnce sync.Once
	idErr  error
	id     int
}

var ErrHsuAccess = errors.New("current user is not in the hsurc file")

// ID returns the current user hsurc identifier. ErrHsuAccess is returned if the current user is not in hsurc.
func (h *Hsu) ID() (int, error) {
	h.idOnce.Do(func() {
		h.id = -1
		hsuPath := internal.MustHsuPath()

		cmd := exec.Command(hsuPath)
		cmd.Path = hsuPath
		cmd.Stderr = os.Stderr // pass through fatal messages
		cmd.Env = make([]string, 0)
		cmd.Dir = container.FHSRoot
		var (
			p         []byte
			exitError *exec.ExitError
		)

		const step = "obtain uid from hsu"
		if p, h.idErr = cmd.Output(); h.idErr == nil {
			h.id, h.idErr = strconv.Atoi(string(p))
			if h.idErr != nil {
				h.idErr = &hst.AppError{Step: step, Err: h.idErr, Msg: "invalid uid string from hsu"}
			}
		} else if errors.As(h.idErr, &exitError) && exitError != nil && exitError.ExitCode() == 1 {
			// hsu prints an error message in this case
			h.idErr = &hst.AppError{Step: step, Err: ErrHsuAccess}
		} else if os.IsNotExist(h.idErr) {
			h.idErr = &hst.AppError{Step: step, Err: os.ErrNotExist,
				Msg: fmt.Sprintf("the setuid helper is missing: %s", hsuPath)}
		}
	})

	return h.id, h.idErr
}

func (h *Hsu) Uid(identity int) (int, error) {
	id, err := h.ID()
	if err == nil {
		return 1000000 + id*10000 + identity, nil
	}
	return id, err
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
