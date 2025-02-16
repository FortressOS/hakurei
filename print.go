package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/state"
)

func printShowSystem(output io.Writer, short bool) {
	t := newPrinter(output)
	defer t.MustFlush()

	info := new(fst.Info)

	// get fid by querying uid of aid 0
	if uid, err := sys.Uid(0); err != nil {
		log.Fatalf("cannot obtain uid from fsu: %v", err)
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
	instance *state.State, config *fst.Config,
	short bool) {
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

	if config.Confinement.Sandbox == nil {
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
		t.Printf(" ID:\t%d (%s)\n", config.Confinement.AppID, config.ID)
	} else {
		t.Printf(" ID:\t%d\n", config.Confinement.AppID)
	}
	t.Printf(" Enablements:\t%s\n", config.Confinement.Enablements.String())
	if len(config.Confinement.Groups) > 0 {
		t.Printf(" Groups:\t%q\n", config.Confinement.Groups)
	}
	t.Printf(" Directory:\t%s\n", config.Confinement.Outer)
	if config.Confinement.Sandbox != nil {
		sandbox := config.Confinement.Sandbox
		if sandbox.Hostname != "" {
			t.Printf(" Hostname:\t%q\n", sandbox.Hostname)
		}
		flags := make([]string, 0, 7)
		writeFlag := func(name string, value bool) {
			if value {
				flags = append(flags, name)
			}
		}
		writeFlag("userns", sandbox.UserNS)
		writeFlag("net", sandbox.Net)
		writeFlag("dev", sandbox.Dev)
		writeFlag("tty", sandbox.NoNewSession)
		writeFlag("mapuid", sandbox.MapRealUID)
		writeFlag("directwl", sandbox.DirectWayland)
		writeFlag("autoetc", sandbox.AutoEtc)
		if len(flags) == 0 {
			flags = append(flags, "none")
		}
		t.Printf(" Flags:\t%s\n", strings.Join(flags, " "))

		etc := sandbox.Etc
		if etc == "" {
			etc = "/etc"
		}
		t.Printf(" Etc:\t%s\n", etc)

		if len(sandbox.Override) > 0 {
			t.Printf(" Overrides:\t%s\n", strings.Join(sandbox.Override, " "))
		}

		// Env           map[string]string   `json:"env"`
		// Link          [][2]string         `json:"symlink"`
	}
	t.Printf(" Command:\t%s\n", strings.Join(config.Command, " "))
	t.Printf("\n")

	if !short {
		if config.Confinement.Sandbox != nil && len(config.Confinement.Sandbox.Filesystem) > 0 {
			t.Printf("Filesystem\n")
			for _, f := range config.Confinement.Sandbox.Filesystem {
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
		if len(config.Confinement.ExtraPerms) > 0 {
			t.Printf("Extra ACL\n")
			for _, p := range config.Confinement.ExtraPerms {
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
	if config.Confinement.SessionBus != nil {
		t.Printf("Session bus\n")
		printDBus(config.Confinement.SessionBus)
		t.Printf("\n")
	}
	if config.Confinement.SystemBus != nil {
		t.Printf("System bus\n")
		printDBus(config.Confinement.SystemBus)
		t.Printf("\n")
	}
}

func printPs(output io.Writer, now time.Time, s state.Store, short bool) {
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

	t.Println("\tInstance\tPID\tApp\tUptime\tEnablements\tCommand")
	for _, e := range exp {
		var (
			es = "(No confinement information)"
			cs = "(No command information)"
			as = "(No configuration information)"
		)
		if e.Config != nil {
			es = e.Config.Confinement.Enablements.String()
			cs = fmt.Sprintf("%q", e.Config.Command)
			as = strconv.Itoa(e.Config.Confinement.AppID)
		}
		t.Printf("\t%s\t%d\t%s\t%s\t%s\t%s\n",
			e.s[:8], e.PID, as, now.Sub(e.Time).Round(time.Second).String(), strings.TrimPrefix(es, ", "), cs)
	}
	t.Println()
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
