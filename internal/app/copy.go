package app

import (
	"io"
	"os"
)

func copyFile(dst, src string) error {
	srcD, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if srcD.Close() != nil {
			// unreachable
			panic("src file closed prematurely")
		}
	}()

	dstD, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if dstD.Close() != nil {
			// unreachable
			panic("dst file closed prematurely")
		}
	}()

	_, err = io.Copy(dstD, srcD)
	return err
}
