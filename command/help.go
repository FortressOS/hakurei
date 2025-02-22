package command

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

var ErrHelp = errors.New("help requested")

func (n *node) PrintHelp() { _ = n.writeHelp() }

func (n *node) writeHelp() error {
	if _, err := fmt.Fprintf(n.out,
		"\nUsage:\t%s [-h | --help]%s COMMAND [OPTIONS]\n",
		strings.Join(append(n.prefix, n.name), " "), &n.suffix,
	); err != nil {
		return err
	}
	if n.child != nil {
		if _, err := fmt.Fprint(n.out, "\nCommands:\n"); err != nil {
			return err
		}
	}

	tw := tabwriter.NewWriter(n.out, 0, 1, 4, ' ', 0)
	if err := n.child.writeCommands(tw); err != nil {
		return err
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	_, err := n.out.Write([]byte{'\n'})
	if err == nil {
		err = ErrHelp
	}
	return err
}

func (n *node) writeCommands(w io.Writer) error {
	if n == nil {
		return nil
	}
	if n.usage != UsageInternal {
		if _, err := fmt.Fprintf(w, "\t%s\t%s\n", n.name, n.usage); err != nil {
			return err
		}
	}
	return n.next.writeCommands(w)
}
