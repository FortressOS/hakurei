package main

import (
	"bytes"
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
	"hakurei.app/internal/env"
	"hakurei.app/internal/outcome"
	"hakurei.app/internal/store"
	"hakurei.app/message"
)

// printShowSystem populates and writes a representation of [hst.Info] to output.
func printShowSystem(output io.Writer, short, flagJSON bool) {
	t := newPrinter(output)
	defer t.MustFlush()

	info := &hst.Info{Version: internal.Version(), User: new(outcome.Hsu).MustID(nil)}
	env.CopyPaths().Copy(&info.Paths, info.User)

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
func printPs(msg message.Msg, output io.Writer, now time.Time, s *store.Store, short, flagJSON bool) {
	f := func(a func(eh *store.EntryHandle)) {
		entries, copyError := s.All()
		for eh := range entries {
			a(eh)
		}
		if err := copyError(); err != nil {
			msg.GetLogger().Println(getMessage("cannot iterate over store:", err))
		}
	}

	if short { // short output requires identifier only
		var identifiers []*hst.ID
		f(func(eh *store.EntryHandle) {
			if _, err := eh.Load(nil); err != nil { // passes through decode error
				msg.GetLogger().Println(getMessage("cannot validate state entry header:", err))
				return
			}
			identifiers = append(identifiers, &eh.ID)
		})
		slices.SortFunc(identifiers, func(a, b *hst.ID) int { return bytes.Compare(a[:], b[:]) })

		if flagJSON {
			encodeJSON(log.Fatal, output, short, identifiers)
		} else {
			for _, id := range identifiers {
				mustPrintln(output, shortIdentifier(id))
			}
		}
		return
	}

	// long output requires full instance state
	var instances []*hst.State
	f(func(eh *store.EntryHandle) {
		var state hst.State
		if _, err := eh.Load(&state); err != nil { // passes through decode error
			msg.GetLogger().Println(getMessage("cannot load state entry:", err))
			return
		}
		instances = append(instances, &state)
	})
	slices.SortFunc(instances, func(a, b *hst.State) int { return bytes.Compare(a.ID[:], b.ID[:]) })

	if flagJSON {
		encodeJSON(log.Fatal, output, short, instances)
		return
	}

	t := newPrinter(output)
	defer t.MustFlush()

	t.Println("\tInstance\tPID\tApplication\tUptime")
	for _, instance := range instances {
		as := "(No configuration information)"
		if instance.Config != nil {
			as = strconv.Itoa(instance.Config.Identity)
			id := instance.Config.ID
			if id == "" {
				id = "app.hakurei." + shortIdentifier(&instance.ID)
			}
			as += " (" + id + ")"
		}
		t.Printf("\t%s\t%d\t%s\t%s\n",
			shortIdentifier(&instance.ID), instance.PID, as, now.Sub(instance.Time).Round(time.Second).String())
	}
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

// getMessage returns a [message.Error] message if available, or err prefixed with fallback otherwise.
func getMessage(fallback string, err error) string {
	if m, ok := message.GetMessage(err); ok {
		return m
	}
	return fmt.Sprintln(fallback, err)
}
