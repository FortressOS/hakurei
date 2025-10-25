package main

import (
	"fmt"
	"io"
	"log"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"hakurei.app/hst"
	"hakurei.app/internal"
	"hakurei.app/internal/app"
	"hakurei.app/internal/app/state"
	"hakurei.app/message"
)

// printShowSystem populates and writes a representation of [hst.Info] to output.
func printShowSystem(output io.Writer, short, flagJSON bool) {
	t := newPrinter(output)
	defer t.MustFlush()

	info := &hst.Info{Version: internal.Version(), User: new(app.Hsu).MustID(nil)}
	app.CopyPaths().Copy(&info.Paths, info.User)

	if flagJSON {
		encodeJSON(log.Fatal, output, short, info)
		return
	}

	t.Printf("Version:\t%s\n", info.Version)
	t.Printf("User:\t%d\n", info.User)
	t.Printf("TempDir:\t%s\n", info.TempDir)
	t.Printf("SharePath:\t%s\n", info.SharePath)
	t.Printf("RuntimePath:\t%s\n", info.RuntimePath)
	t.Printf("RunDirPath:\t%s\n", info.RunDirPath)
}

// printShowInstance writes a representation of [hst.State] or [hst.Config] to output.
func printShowInstance(
	output io.Writer, now time.Time,
	instance *hst.State, config *hst.Config,
	short, flagJSON bool) (valid bool) {
	valid = true

	if flagJSON {
		if instance != nil {
			encodeJSON(log.Fatal, output, short, instance)
		} else {
			encodeJSON(log.Fatal, output, short, config)
		}
		return
	}

	t := newPrinter(output)
	defer t.MustFlush()

	if err := config.Validate(); err != nil {
		valid = false
		if m, ok := message.GetMessage(err); ok {
			mustPrint(output, "Error: "+m+"!\n\n")
		}
	}

	if instance != nil {
		t.Printf("State\n")
		t.Printf(" Instance:\t%s (%d -> %d)\n", instance.ID.String(), instance.PID, instance.ShimPID)
		t.Printf(" Uptime:\t%s\n", now.Sub(instance.Time).Round(time.Second).String())
		t.Printf("\n")
	}

	t.Printf("App\n")
	if config.ID != "" {
		t.Printf(" Identity:\t%d (%s)\n", config.Identity, config.ID)
	} else {
		t.Printf(" Identity:\t%d\n", config.Identity)
	}
	t.Printf(" Enablements:\t%s\n", config.Enablements.Unwrap().String())
	if len(config.Groups) > 0 {
		t.Printf(" Groups:\t%s\n", strings.Join(config.Groups, ", "))
	}
	if config.Container != nil {
		params := config.Container
		if params.Home != nil {
			t.Printf(" Home:\t%s\n", params.Home)
		}
		if params.Hostname != "" {
			t.Printf(" Hostname:\t%s\n", params.Hostname)
		}
		flags := params.Flags.String()

		// this is included in the upper hst.Config struct but is relevant here
		const flagDirectWayland = "directwl"
		if config.DirectWayland {
			// hardcoded value when every flag is unset
			if flags == "none" {
				flags = flagDirectWayland
			} else {
				flags += ", " + flagDirectWayland
			}
		}
		t.Printf(" Flags:\t%s\n", flags)

		if params.Path != nil {
			t.Printf(" Path:\t%s\n", params.Path)
		}
		if len(params.Args) > 0 {
			t.Printf(" Arguments:\t%s\n", strings.Join(params.Args, " "))
		}
	}
	t.Printf("\n")

	if !short {
		if config.Container != nil && len(config.Container.Filesystem) > 0 {
			t.Printf("Filesystem\n")
			for _, f := range config.Container.Filesystem {
				if !f.Valid() {
					valid = false
					t.Println(" <invalid>")
					continue
				}
				t.Printf(" %s\n", f)
			}
			t.Printf("\n")
		}
		if len(config.ExtraPerms) > 0 {
			t.Printf("Extra ACL\n")
			for i := range config.ExtraPerms {
				t.Printf(" %s\n", config.ExtraPerms[i].String())
			}
			t.Printf("\n")
		}
	}

	printDBus := func(c *hst.BusConfig) {
		t.Printf(" Filter:\t%v\n", c.Filter)
		if len(c.See) > 0 {
			t.Printf(" See:\t%q\n", c.See)
		}
		if len(c.Talk) > 0 {
			t.Printf(" Talk:\t%q\n", c.Talk)
		}
		if len(c.Own) > 0 {
			t.Printf(" Own:\t%q\n", c.Own)
		}
		if len(c.Call) > 0 {
			t.Printf(" Call:\t%q\n", c.Call)
		}
		if len(c.Broadcast) > 0 {
			t.Printf(" Broadcast:\t%q\n", c.Broadcast)
		}
	}
	if config.SessionBus != nil {
		t.Printf("Session bus\n")
		printDBus(config.SessionBus)
		t.Printf("\n")
	}
	if config.SystemBus != nil {
		t.Printf("System bus\n")
		printDBus(config.SystemBus)
		t.Printf("\n")
	}

	return
}

// printPs writes a representation of active instances to output.
func printPs(output io.Writer, now time.Time, s state.Store, short, flagJSON bool) {
	var entries map[hst.ID]*hst.State
	if e, err := state.Join(s); err != nil {
		log.Fatalf("cannot join store: %v", err)
	} else {
		entries = e
	}

	if !short && flagJSON {
		es := make(map[string]*hst.State, len(entries))
		for id, instance := range entries {
			es[id.String()] = instance
		}
		encodeJSON(log.Fatal, output, short, es)
		return
	}

	// sort state entries by id string to ensure consistency between runs
	exp := make([]*expandedStateEntry, 0, len(entries))
	for id, instance := range entries {
		// gracefully skip nil states
		if instance == nil {
			log.Printf("got invalid state entry %s", id.String())
			continue
		}

		// gracefully skip inconsistent states
		if id != instance.ID {
			log.Printf("possible store corruption: entry %s has id %s",
				id.String(), instance.ID.String())
			continue
		}
		exp = append(exp, &expandedStateEntry{s: id.String(), State: instance})
	}
	slices.SortFunc(exp, func(a, b *expandedStateEntry) int { return a.Time.Compare(b.Time) })

	if short {
		if flagJSON {
			v := make([]string, len(exp))
			for i, e := range exp {
				v[i] = e.s
			}
			encodeJSON(log.Fatal, output, short, v)
		} else {
			for _, e := range exp {
				mustPrintln(output, shortIdentifierString(e.s))
			}
		}
		return
	}

	t := newPrinter(output)
	defer t.MustFlush()

	t.Println("\tInstance\tPID\tApplication\tUptime")
	for _, e := range exp {
		if len(e.s) != 1<<5 {
			// unreachable
			log.Printf("possible store corruption: invalid instance string %s", e.s)
			continue
		}

		as := "(No configuration information)"
		if e.Config != nil {
			as = strconv.Itoa(e.Config.Identity)
			id := e.Config.ID
			if id == "" {
				id = "app.hakurei." + shortIdentifierString(e.s)
			}
			as += " (" + id + ")"
		}
		t.Printf("\t%s\t%d\t%s\t%s\n",
			shortIdentifierString(e.s), e.PID, as, now.Sub(e.Time).Round(time.Second).String())
	}
}

// expandedStateEntry stores [hst.State] alongside a string representation of its [hst.ID].
type expandedStateEntry struct {
	s string
	*hst.State
}

// newPrinter returns a configured, wrapped [tabwriter.Writer].
func newPrinter(output io.Writer) *tp { return &tp{tabwriter.NewWriter(output, 0, 1, 4, ' ', 0)} }

// tp wraps [tabwriter.Writer] to provide additional formatting methods.
type tp struct{ *tabwriter.Writer }

// Printf calls [fmt.Fprintf] on the underlying [tabwriter.Writer].
func (p *tp) Printf(format string, a ...any) {
	if _, err := fmt.Fprintf(p, format, a...); err != nil {
		log.Fatalf("cannot write to tabwriter: %v", err)
	}
}

// Println calls [fmt.Fprintln] on the underlying [tabwriter.Writer].
func (p *tp) Println(a ...any) {
	if _, err := fmt.Fprintln(p, a...); err != nil {
		log.Fatalf("cannot write to tabwriter: %v", err)
	}
}

// MustFlush calls the Flush method of [tabwriter.Writer] and calls [log.Fatalf] on a non-nil error.
func (p *tp) MustFlush() {
	if err := p.Writer.Flush(); err != nil {
		log.Fatalf("cannot flush tabwriter: %v", err)
	}
}

func mustPrint(output io.Writer, a ...any) {
	if _, err := fmt.Fprint(output, a...); err != nil {
		log.Fatalf("cannot print: %v", err)
	}
}
func mustPrintln(output io.Writer, a ...any) {
	if _, err := fmt.Fprintln(output, a...); err != nil {
		log.Fatalf("cannot print: %v", err)
	}
}
