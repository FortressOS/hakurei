package main

import (
	"encoding/json"
	"fmt"
	direct "os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"git.gensokyo.uk/security/fortify/dbus"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/state"
)

func printShowSystem(short bool) {
	info := new(fst.Info)

	// get fid by querying uid of aid 0
	if uid, err := os.Uid(0); err != nil {
		fmsg.Fatalf("cannot obtain uid from fsu: %v", err)
	} else {
		info.User = (uid / 10000) - 100
	}

	if flagJSON {
		printJSON(info)
		return
	}

	w := tabwriter.NewWriter(direct.Stdout, 0, 1, 4, ' ', 0)

	fmt.Fprintf(w, "User:\t%d\n", info.User)

	if err := w.Flush(); err != nil {
		fmsg.Fatalf("cannot flush tabwriter: %v", err)
	}
}

func printShowInstance(instance *state.State, config *fst.Config, short bool) {
	if flagJSON {
		if instance != nil {
			printJSON(instance)
		} else {
			printJSON(config)
		}
		return
	}

	now := time.Now().UTC()
	w := tabwriter.NewWriter(direct.Stdout, 0, 1, 4, ' ', 0)

	if instance != nil {
		fmt.Fprintf(w, "State\n")
		fmt.Fprintf(w, " Instance:\t%s (%d)\n", instance.ID.String(), instance.PID)
		fmt.Fprintf(w, " Uptime:\t%s\n", now.Sub(instance.Time).Round(time.Second).String())
		fmt.Fprintf(w, "\n")
	}

	fmt.Fprintf(w, "App\n")
	if config.ID != "" {
		fmt.Fprintf(w, " ID:\t%d (%s)\n", config.Confinement.AppID, config.ID)
	} else {
		fmt.Fprintf(w, " ID:\t%d\n", config.Confinement.AppID)
	}
	fmt.Fprintf(w, " Enablements:\t%s\n", config.Confinement.Enablements.String())
	if len(config.Confinement.Groups) > 0 {
		fmt.Fprintf(w, " Groups:\t%q\n", config.Confinement.Groups)
	}
	fmt.Fprintf(w, " Directory:\t%s\n", config.Confinement.Outer)
	if config.Confinement.Sandbox != nil {
		sandbox := config.Confinement.Sandbox
		if sandbox.Hostname != "" {
			fmt.Fprintf(w, " Hostname:\t%q\n", sandbox.Hostname)
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
		fmt.Fprintf(w, " Flags:\t%s\n", strings.Join(flags, " "))

		etc := sandbox.Etc
		if etc == "" {
			etc = "/etc"
		}
		fmt.Fprintf(w, " Etc:\t%s\n", etc)

		if len(sandbox.Override) > 0 {
			fmt.Fprintf(w, " Overrides:\t%s\n", strings.Join(sandbox.Override, " "))
		}

		// Env           map[string]string   `json:"env"`
		// Link          [][2]string         `json:"symlink"`
	} else {
		// this gets printed before everything else
		fmt.Println("WARNING: current configuration uses permissive defaults!")
	}
	fmt.Fprintf(w, " Command:\t%s\n", strings.Join(config.Command, " "))
	fmt.Fprintf(w, "\n")

	if !short {
		if config.Confinement.Sandbox != nil && len(config.Confinement.Sandbox.Filesystem) > 0 {
			fmt.Fprintf(w, "Filesystem\n")
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
				fmt.Fprintf(w, "%s\n", expr.String())
			}
			fmt.Fprintf(w, "\n")
		}
		if len(config.Confinement.ExtraPerms) > 0 {
			fmt.Fprintf(w, "Extra ACL\n")
			for _, p := range config.Confinement.ExtraPerms {
				if p == nil {
					continue
				}
				fmt.Fprintf(w, " %s\n", p.String())
			}
			fmt.Fprintf(w, "\n")
		}
	}

	printDBus := func(c *dbus.Config) {
		fmt.Fprintf(w, " Filter:\t%v\n", c.Filter)
		if len(c.See) > 0 {
			fmt.Fprintf(w, " See:\t%q\n", c.See)
		}
		if len(c.Talk) > 0 {
			fmt.Fprintf(w, " Talk:\t%q\n", c.Talk)
		}
		if len(c.Own) > 0 {
			fmt.Fprintf(w, " Own:\t%q\n", c.Own)
		}
		if len(c.Call) > 0 {
			fmt.Fprintf(w, " Call:\t%q\n", c.Call)
		}
		if len(c.Broadcast) > 0 {
			fmt.Fprintf(w, " Broadcast:\t%q\n", c.Broadcast)
		}
	}
	if config.Confinement.SessionBus != nil {
		fmt.Fprintf(w, "Session bus\n")
		printDBus(config.Confinement.SessionBus)
		fmt.Fprintf(w, "\n")
	}
	if config.Confinement.SystemBus != nil {
		fmt.Fprintf(w, "System bus\n")
		printDBus(config.Confinement.SystemBus)
		fmt.Fprintf(w, "\n")
	}

	if err := w.Flush(); err != nil {
		fmsg.Fatalf("cannot flush tabwriter: %v", err)
	}
}

func printPs(short bool) {
	now := time.Now().UTC()

	var entries state.Entries
	s := state.NewMulti(os.Paths().RunDirPath)
	if e, err := state.Join(s); err != nil {
		fmsg.Fatalf("cannot join store: %v", err)
	} else {
		entries = e
	}
	if err := s.Close(); err != nil {
		fmsg.Printf("cannot close store: %v", err)
	}

	if flagJSON {
		es := make(map[string]*state.State, len(entries))
		for id, instance := range entries {
			es[id.String()] = instance
		}
		printJSON(es)
		return
	}

	// sort state entries by id string to ensure consistency between runs
	exp := make([]*expandedStateEntry, 0, len(entries))
	for id, instance := range entries {
		// gracefully skip nil states
		if instance == nil {
			fmsg.Printf("got invalid state entry %s", id.String())
			continue
		}

		// gracefully skip inconsistent states
		if id != instance.ID {
			fmt.Printf("possible store corruption: entry %s has id %s",
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
			printJSON(v)
		} else {
			for _, e := range exp {
				fmt.Println(e.s[:8])
			}
		}
		return
	}

	// buffer output to reduce terminal activity
	w := tabwriter.NewWriter(direct.Stdout, 0, 1, 4, ' ', 0)
	fmt.Fprintln(w, "\tInstance\tPID\tApp\tUptime\tEnablements\tCommand")
	for _, e := range exp {
		printInstance(w, e, now)
	}
	if err := w.Flush(); err != nil {
		fmsg.Fatalf("cannot flush tabwriter: %v", err)
	}
}

type expandedStateEntry struct {
	s string
	*state.State
}

func printInstance(w *tabwriter.Writer, e *expandedStateEntry, now time.Time) {
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
	fmt.Fprintf(w, "\t%s\t%d\t%s\t%s\t%s\t%s\n",
		e.s[:8], e.PID, as, now.Sub(e.Time).Round(time.Second).String(), strings.TrimPrefix(es, ", "), cs)
}

func printJSON(v any) {
	encoder := json.NewEncoder(direct.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		fmsg.Fatalf("cannot serialise: %v", err)
		panic("unreachable")
	}
}
