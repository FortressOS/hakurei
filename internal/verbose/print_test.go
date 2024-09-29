package verbose_test

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"git.ophivana.moe/cat/fortify/internal/verbose"
)

const (
	testVerbose = "GO_TEST_VERBOSE"
	wantStdout  = "fortify: println\nfortify: printf"
)

func TestPrinter(t *testing.T) {
	switch os.Getenv(testVerbose) {
	case "0":
		verbose.Set(false)
	case "1":
		verbose.Set(true)
	default:
		return
	}

	verbose.Println("println")
	verbose.Printf("%s", "printf")
}

func TestPrintf_Println(t *testing.T) {
	testPrintfPrintln(t, false)
	testPrintfPrintln(t, true)

	// make -cover happy
	stdout := os.Stdout
	t.Cleanup(func() {
		os.Stdout = stdout
	})
	os.Stdout = nil
	verbose.Set(true)
	verbose.Printf("")
	verbose.Println()
}

func testPrintfPrintln(t *testing.T, v bool) {
	t.Run("start verbose printer with verbose "+strconv.FormatBool(v), func(t *testing.T) {
		stdout, stderr := new(strings.Builder), new(strings.Builder)
		stdout.Grow(len(wantStdout))
		cmd := exec.Command(os.Args[0], "-test.run=TestPrinter")
		cmd.Stdout, cmd.Stderr = stdout, stderr
		if v {
			cmd.Env = append(cmd.Env, testVerbose+"=1")
		} else {
			cmd.Env = append(cmd.Env, testVerbose+"=0")
		}
		if err := cmd.Run(); err != nil {
			panic("cannot run printer process: " + err.Error() + " stderr: " + stderr.String())
		}

		if got := stdout.String(); strings.Contains(got, wantStdout) != v {
			t.Errorf("Print: got %v; want %t",
				got, v)
		}
	})
}
