package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"hakurei.app/hst"
	"hakurei.app/internal/app/state"
	"hakurei.app/internal/hlog"
	"hakurei.app/system/dbus"
)

func printShowSystem(output io.Writer, short, flagJSON bool) {
	t := newPrinter(output)
	defer t.MustFlush()

	info := new(hst.Info)

	// get fid by querying uid of aid 0
	if uid, err := std.Uid(0); err != nil {
		hlog.PrintBaseError(err, "cannot obtain uid from setuid wrapper:")
		os.Exit(1)
	} else {
		info.User = (uid / 10000) - 100
	}

	if flagJSON {
		printJSON(output, short, info)
		return
	}

	t.Printf("User:\t%d\n", info.User)
}

func printShowInstance(
	output io.Writer, now time.Time,
	instance *state.State, config *hst.Config,
	short, flagJSON bool) {
	if flagJSON {
		if instance != nil {
			printJSON(output, short, instance)
		} else {
			printJSON(output, short, config)
		}
		return
	}

	t := newPrinter(output)
	defer t.MustFlush()

	if config.Container == nil {
		mustPrint(output, "Warning: this configuration uses permissive defaults!\n\n")
	}

	if instance != nil {
		t.Printf("State\n")
		t.Printf(" Instance:\t%s (%d)\n", instance.ID.String(), instance.PID)
		t.Printf(" Uptime:\t%s\n", now.Sub(instance.Time).Round(time.Second).String())
		t.Printf("\n")
	}

	t.Printf("App\n")
	if config.ID != "" {
		t.Printf(" Identity:\t%d (%s)\n", config.Identity, config.ID)
	} else {
		t.Printf(" Identity:\t%d\n", config.Identity)
	}
	t.Printf(" Enablements:\t%s\n", config.Enablements.String())
	if len(config.Groups) > 0 {
		t.Printf(" Groups:\t%s\n", strings.Join(config.Groups, ", "))
	}
	if config.Data != "" {
		t.Printf(" Data:\t%s\n", config.Data)
	}
	if config.Container != nil {
		container := config.Container
		if container.Hostname != "" {
			t.Printf(" Hostname:\t%s\n", container.Hostname)
		}
		flags := make([]string, 0, 7)
		writeFlag := func(name string, value bool) {
			if value {
				flags = append(flags, name)
			}
		}
		writeFlag("userns", container.Userns)
		writeFlag("devel", container.Devel)
		writeFlag("net", container.Net)
		writeFlag("device", container.Device)
		writeFlag("tty", container.Tty)
		writeFlag("mapuid", container.MapRealUID)
		writeFlag("directwl", config.DirectWayland)
		writeFlag("autoetc", container.AutoEtc)
		if len(flags) == 0 {
			flags = append(flags, "none")
		}
		t.Printf(" Flags:\t%s\n", strings.Join(flags, " "))

		etc := container.Etc
		if etc == "" {
			etc = "/etc"
		}
		t.Printf(" Etc:\t%s\n", etc)

		if len(container.Cover) > 0 {
			t.Printf(" Cover:\t%s\n", strings.Join(container.Cover, " "))
		}

		t.Printf(" Path:\t%s\n", config.Path)
	}
	if len(config.Args) > 0 {
		t.Printf(" Arguments:\t%s\n", strings.Join(config.Args, " "))
	}
	t.Printf("\n")

	if !short {
		if config.Container != nil && len(config.Container.Filesystem) > 0 {
			t.Printf("Filesystem\n")
			for _, f := range config.Container.Filesystem {
				if f == nil {
					continue
				}

				expr := new(strings.Builder)
				expr.Grow(3 + len(f.Src) + 1 + len(f.Dst))

				if f.Device {
					expr.WriteString(" d")
				} else if f.Write {
					expr.WriteString(" w")
				} else {
					expr.WriteString(" ")
				}
				if f.Must {
					expr.WriteString("*")
				} else {
					expr.WriteString("+")
				}
				expr.WriteString(f.Src)
				if f.Dst != "" {
					expr.WriteString(":" + f.Dst)
				}
				t.Printf("%s\n", expr.String())
			}
			t.Printf("\n")
		}
		if len(config.ExtraPerms) > 0 {
			t.Printf("Extra ACL\n")
			for _, p := range config.ExtraPerms {
				if p == nil {
					continue
				}
				t.Printf(" %s\n", p.String())
			}
			t.Printf("\n")
		}
	}

	printDBus := func(c *dbus.Config) {
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
}

func printPs(output io.Writer, now time.Time, s state.Store, short, flagJSON bool) {
	var entries state.Entries
	if e, err := state.Join(s); err != nil {
		log.Fatalf("cannot join store: %v", err)
	} else {
		entries = e
	}
	if err := s.Close(); err != nil {
		log.Printf("cannot close store: %v", err)
	}

	if !short && flagJSON {
		es := make(map[string]*state.State, len(entries))
		for id, instance := range entries {
			es[id.String()] = instance
		}
		printJSON(output, short, es)
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
			printJSON(output, short, v)
		} else {
			for _, e := range exp {
				mustPrintln(output, e.s[:8])
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
				id = "app.hakurei." + e.s[:8]
			}
			as += " (" + id + ")"
		}
		t.Printf("\t%s\t%d\t%s\t%s\n",
			e.s[:8], e.PID, as, now.Sub(e.Time).Round(time.Second).String())
	}
}

type expandedStateEntry struct {
	s string
	*state.State
}

func printJSON(output io.Writer, short bool, v any) {
	encoder := json.NewEncoder(output)
	if !short {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(v); err != nil {
		log.Fatalf("cannot serialise: %v", err)
	}
}

func newPrinter(output io.Writer) *tp { return &tp{tabwriter.NewWriter(output, 0, 1, 4, ' ', 0)} }

type tp struct{ *tabwriter.Writer }

func (p *tp) Printf(format string, a ...any) {
	if _, err := fmt.Fprintf(p, format, a...); err != nil {
		log.Fatalf("cannot write to tabwriter: %v", err)
	}
}
func (p *tp) Println(a ...any) {
	if _, err := fmt.Fprintln(p, a...); err != nil {
		log.Fatalf("cannot write to tabwriter: %v", err)
	}
}
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
