package helper

import (
	"testing"
)

func Test_pipes_pipe_mustClosePipes(t *testing.T) {
	p := new(pipes)

	t.Run("pipe without args", func(t *testing.T) {
		defer func() {
			wantPanic := "attempted to pipe without args"
			if r := recover(); r != wantPanic {
				t.Errorf("pipe() panic = %v, wantPanic %v",
					r, wantPanic)
			}
		}()
		_ = p.pipe()
	})

	p.args = MustNewCheckedArgs(make([]string, 0))
	t.Run("obtain pipes", func(t *testing.T) {
		if err := p.pipe(); err != nil {
			t.Errorf("pipe() error = %v",
				err)
			return
		}
	})

	t.Run("pipe twice", func(t *testing.T) {
		defer func() {
			wantPanic := "attempted to pipe twice"
			if r := recover(); r != wantPanic {
				t.Errorf("pipe() panic = %v, wantPanic %v",
					r, wantPanic)
			}
		}()
		_ = p.pipe()
	})

	p.mustClosePipes()
}
