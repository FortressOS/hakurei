package bwrap

import (
	"context"
	"encoding/gob"
	"os"
	"strconv"

	"git.gensokyo.uk/security/fortify/helper/proc"
)

func init() {
	gob.Register(new(pairF))
	gob.Register(new(stringF))
}

type pairF [3]string

func (p *pairF) Path() string          { return p[2] }
func (p *pairF) Len() int              { return len(p) }
func (p *pairF) Append(args *[]string) { *args = append(*args, p[0], p[1], p[2]) }

type stringF [2]string

func (s stringF) Path() string          { return s[1] }
func (s stringF) Len() int              { return len(s) /* compiler replaces this with 2 */ }
func (s stringF) Append(args *[]string) { *args = append(*args, s[0], s[1]) }

func newFile(name string, f *os.File) FDBuilder { return &fileF{name: name, file: f} }

type fileF struct {
	name string
	file *os.File
	proc.BaseFile
}

func (f *fileF) ErrCount() int                                  { return 0 }
func (f *fileF) Fulfill(_ context.Context, _ func(error)) error { f.Set(f.file); return nil }

func (f *fileF) Len() int {
	if f.file == nil {
		return 0
	}
	return 2
}

func (f *fileF) Append(args *[]string) {
	if f.file == nil {
		return
	}
	*args = append(*args, f.name, strconv.Itoa(int(f.Fd())))
}
