package command_test

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"

	"git.gensokyo.uk/security/hakurei/command"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name      string
		buildTree func(wout, wlog io.Writer) command.Command
		args      []string
		want      string
		wantLog   string
		wantErr   error
	}{
		{
			"d=0 empty sub",
			func(wout, wlog io.Writer) command.Command { return command.New(wout, newLogFunc(wlog), "root", nil) },
			[]string{""},
			"", "test: \"root\" has no subcommands\n", command.ErrEmptyTree,
		},
		{
			"d=0 empty sub garbage",
			func(wout, wlog io.Writer) command.Command { return command.New(wout, newLogFunc(wlog), "root", nil) },
			[]string{"a", "b", "c", "d"},
			"", "test: \"root\" has no subcommands\n", command.ErrEmptyTree,
		},
		{
			"d=0 no match",
			buildTestCommand,
			[]string{"nonexistent"},
			"", "test: \"nonexistent\" is not a valid command\n", command.ErrNoMatch,
		},
		{
			"d=0 direct error",
			buildTestCommand,
			[]string{"error"},
			"", "", errSuccess,
		},
		{
			"d=0 direct error garbage",
			buildTestCommand,
			[]string{"error", "0", "1", "2"},
			"", "", errSuccess,
		},
		{
			"d=0 direct success out of order",
			buildTestCommand,
			[]string{"succeed"},
			"", "", nil,
		},
		{
			"d=0 direct success output",
			buildTestCommand,
			[]string{"print", "0", "1", "2"},
			"012", "", nil,
		},
		{
			"d=0 out of order string flag",
			buildTestCommand,
			[]string{"string", "--string", "64d3b4b7b21788585845060e2199a78f"},
			"flag provided but not defined: -string\n\nUsage:\ttest string [-h | --help] COMMAND [OPTIONS]\n\n", "",
			errors.New("flag provided but not defined: -string"),
		},
		{
			"d=0 string flag",
			buildTestCommand,
			[]string{"--string", "64d3b4b7b21788585845060e2199a78f", "string"},
			"64d3b4b7b21788585845060e2199a78f", "", nil,
		},
		{
			"d=0 int flag",
			buildTestCommand,
			[]string{"--int", "2147483647", "int"},
			"2147483647", "", nil,
		},
		{
			"d=0 repeat flag",
			buildTestCommand,
			[]string{"--repeat", "0", "--repeat", "1", "--repeat", "2", "--repeat", "3", "--repeat", "4", "repeat"},
			"[0 1 2 3 4]", "", nil,
		},
		{
			"d=0 bool flag",
			buildTestCommand,
			[]string{"-v", "succeed"},
			"", "test: verbose\n", nil,
		},
		{
			"d=0 bool flag early error",
			buildTestCommand,
			[]string{"--fail", "succeed"},
			"", "", errSuccess,
		},

		{
			"d=1 empty sub",
			buildTestCommand,
			[]string{"empty"},
			"", "test: \"empty\" has no subcommands\n", command.ErrEmptyTree,
		},
		{
			"d=1 empty sub garbage",
			buildTestCommand,
			[]string{"empty", "a", "b", "c", "d"},
			"", "test: \"empty\" has no subcommands\n", command.ErrEmptyTree,
		},
		{
			"d=1 empty sub help",
			buildTestCommand,
			[]string{"empty", "-h"},
			"\nUsage:\ttest empty [-h | --help] COMMAND [OPTIONS]\n\n", "", flag.ErrHelp,
		},
		{
			"d=1 no match",
			buildTestCommand,
			[]string{"join", "23aa3bb0", "34986782", "d8859355", "cd9ac317", ", "},
			"", "test: \"23aa3bb0\" is not a valid command\n", command.ErrNoMatch,
		},
		{
			"d=1 direct success out",
			buildTestCommand,
			[]string{"join", "out", "23aa3bb0", "34986782", "d8859355", "cd9ac317", ", "},
			"23aa3bb0, 34986782, d8859355, cd9ac317", "", nil,
		},
		{
			"d=1 direct success log",
			buildTestCommand,
			[]string{"join", "log", "23aa3bb0", "34986782", "d8859355", "cd9ac317", ", "},
			"", "test: 23aa3bb0, 34986782, d8859355, cd9ac317\n", nil,
		},

		{
			"d=4 empty sub",
			buildTestCommand,
			[]string{"deep", "d=2", "d=3", "d=4"},
			"", "test: \"d=4\" has no subcommands\n", command.ErrEmptyTree},

		{
			"d=0 help",
			buildTestCommand,
			[]string{},
			`
Usage:	test [-h | --help] [-v] [--fail] [--string <value>] [--int <int>] [--repeat <value>] COMMAND [OPTIONS]

Commands:
    error      return an error
    print      wraps Fprint
    string     print string passed by flag
    int        print int passed by flag
    repeat     print repeated values passed by flag
    empty      empty subcommand
    join       wraps strings.Join
    succeed    this command succeeds
    deep       top level of command tree with various levels

`, "", command.ErrHelp,
		},
		{
			"d=0 help flag",
			buildTestCommand,
			[]string{"-h"},
			`
Usage:	test [-h | --help] [-v] [--fail] [--string <value>] [--int <int>] [--repeat <value>] COMMAND [OPTIONS]

Commands:
    error      return an error
    print      wraps Fprint
    string     print string passed by flag
    int        print int passed by flag
    repeat     print repeated values passed by flag
    empty      empty subcommand
    join       wraps strings.Join
    succeed    this command succeeds
    deep       top level of command tree with various levels

Flags:
  -fail
    	fail early
  -int int
    	store value for the "int" command (default -1)
  -repeat value
    	store value for the "repeat" command
  -string string
    	store value for the "string" command (default "default")
  -v	verbose output

`, "", flag.ErrHelp,
		},

		{
			"d=1 help",
			buildTestCommand,
			[]string{"join"},
			`
Usage:	test join [-h | --help] COMMAND [OPTIONS]

Commands:
    out    write result to wout
    log    log result to wlog

`, "", command.ErrHelp,
		},
		{
			"d=1 help flag",
			buildTestCommand,
			[]string{"join", "-h"},
			`
Usage:	test join [-h | --help] COMMAND [OPTIONS]

Commands:
    out    write result to wout
    log    log result to wlog

`, "", flag.ErrHelp,
		},

		{
			"d=2 help",
			buildTestCommand,
			[]string{"deep", "d=2"},
			`
Usage:	test deep d=2 [-h | --help] COMMAND [OPTIONS]

Commands:
    d=3    relative third level

`, "", command.ErrHelp,
		},
		{
			"d=2 help flag",
			buildTestCommand,
			[]string{"deep", "d=2", "-h"},
			`
Usage:	test deep d=2 [-h | --help] COMMAND [OPTIONS]

Commands:
    d=3    relative third level

`, "", flag.ErrHelp,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wout, wlog := new(bytes.Buffer), new(bytes.Buffer)
			c := tc.buildTree(wout, wlog)

			if err := c.Parse(tc.args); !errors.Is(err, tc.wantErr) {
				t.Errorf("Parse: error = %v; wantErr %v", err, tc.wantErr)
			}
			if got := wout.String(); got != tc.want {
				t.Errorf("Parse: %s want %s", got, tc.want)
			}
			if gotLog := wlog.String(); gotLog != tc.wantLog {
				t.Errorf("Parse: log = %s wantLog %s", gotLog, tc.wantLog)
			}
		})
	}
}

var (
	errJoinLen = errors.New("not enough arguments to join")
	errSuccess = errors.New("success")
)

func buildTestCommand(wout, wlog io.Writer) (c command.Command) {
	var (
		flagVerbose bool
		flagFail    bool

		flagString string
		flagInt    int
		flagRepeat command.RepeatableFlag
	)

	logf := newLogFunc(wlog)
	c = command.New(wout, logf, "test", func([]string) error {
		if flagVerbose {
			logf("verbose")
		}
		if flagFail {
			return errSuccess
		}
		return nil
	}).
		Flag(&flagVerbose, "v", command.BoolFlag(false), "verbose output").
		Flag(&flagFail, "fail", command.BoolFlag(false), "fail early").
		Command("error", "return an error", func([]string) error {
			return errSuccess
		}).
		Command("print", "wraps Fprint", func(args []string) error {
			a := make([]any, len(args))
			for i, v := range args {
				a[i] = v
			}
			_, err := fmt.Fprint(wout, a...)
			return err
		}).
		Flag(&flagString, "string", command.StringFlag("default"), "store value for the \"string\" command").
		Command("string", "print string passed by flag", func(args []string) error { _, err := fmt.Fprint(wout, flagString); return err }).
		Flag(&flagInt, "int", command.IntFlag(-1), "store value for the \"int\" command").
		Command("int", "print int passed by flag", func(args []string) error { _, err := fmt.Fprint(wout, flagInt); return err }).
		Flag(nil, "repeat", &flagRepeat, "store value for the \"repeat\" command").
		Command("repeat", "print repeated values passed by flag", func(args []string) error { _, err := fmt.Fprint(wout, flagRepeat); return err })

	c.New("empty", "empty subcommand")
	c.New("hidden", command.UsageInternal)

	c.New("join", "wraps strings.Join").
		Command("out", "write result to wout", func(args []string) error {
			if len(args) == 0 {
				return errJoinLen
			}
			_, err := fmt.Fprint(wout, strings.Join(args[:len(args)-1], args[len(args)-1]))
			return err
		}).
		Command("log", "log result to wlog", func(args []string) error {
			if len(args) == 0 {
				return errJoinLen
			}
			logf("%s", strings.Join(args[:len(args)-1], args[len(args)-1]))
			return nil
		})

	c.Command("succeed", "this command succeeds", func([]string) error { return nil })

	c.New("deep", "top level of command tree with various levels").
		New("d=2", "relative second level").
		New("d=3", "relative third level").
		New("d=4", "relative fourth level")

	return
}

func newLogFunc(w io.Writer) command.LogFunc { return log.New(w, "test: ", 0).Printf }
