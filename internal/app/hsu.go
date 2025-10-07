package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"hakurei.app/container"
	"hakurei.app/container/fhs"
	"hakurei.app/hst"
)

// Hsu caches responses from cmd/hsu.
type Hsu struct {
	idOnce sync.Once
	idErr  error
	id     int

	kOnce sync.Once
	k     syscallDispatcher
}

var ErrHsuAccess = errors.New("current user is not in the hsurc file")

// ensureDispatcher ensures Hsu.k is not nil.
func (h *Hsu) ensureDispatcher() {
	h.kOnce.Do(func() {
		if h.k == nil {
			h.k = direct{}
		}
	})
}

// ID returns the current user hsurc identifier. ErrHsuAccess is returned if the current user is not in hsurc.
func (h *Hsu) ID() (int, error) {
	h.ensureDispatcher()
	h.idOnce.Do(func() {
		h.id = -1
		hsuPath := h.k.mustHsuPath().String()

		cmd := exec.Command(hsuPath)
		cmd.Path = hsuPath
		cmd.Stderr = os.Stderr // pass through fatal messages
		cmd.Env = make([]string, 0)
		cmd.Dir = fhs.Root
		var (
			p         []byte
			exitError *exec.ExitError
		)

		const step = "obtain uid from hsu"
		if p, h.idErr = h.k.cmdOutput(cmd); h.idErr == nil {
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

// MustID calls [Hsu.ID] and terminates on error.
func (h *Hsu) MustID() int { return h.MustIDMsg(nil) }

// MustIDMsg implements MustID with a custom [container.Msg].
func (h *Hsu) MustIDMsg(msg container.Msg) int {
	id, err := h.ID()
	if err == nil {
		return id
	}

	const fallback = "cannot retrieve user id from setuid wrapper:"
	if errors.Is(err, ErrHsuAccess) {
		if msg != nil {
			msg.Verbose("*"+fallback, err)
		}
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

// HsuUid returns target uid for the stable hsu uid format.
// No bounds check is performed, a value retrieved from hsu is expected.
func HsuUid(id, identity int) int { return 1000000 + id*10000 + identity }
