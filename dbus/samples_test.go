package dbus_test

import (
	"sync"

	"git.ophivana.moe/cat/fortify/dbus"
)

var samples = []dbusTestCase{
	{
		"org.chromium.Chromium", &dbus.Config{
			See:       nil,
			Talk:      []string{"org.freedesktop.Notifications", "org.freedesktop.FileManager1", "org.freedesktop.ScreenSaver"},
			Own:       []string{"org.chromium.Chromium.*", "org.mpris.MediaPlayer2.org.chromium.Chromium.*", "org.mpris.MediaPlayer2.chromium.*"},
			Call:      map[string]string{"org.freedesktop.portal.*": "*"},
			Broadcast: map[string]string{"org.freedesktop.portal.*": "@/org/freedesktop/portal/*"},
			Log:       false,
			Filter:    true,
		}, false,
		[2]string{"unix:path=/run/user/1971/bus", "/tmp/fortify.1971/12622d846cc3fe7b4c10359d01f0eb47/bus"},
		[]string{
			"unix:path=/run/user/1971/bus",
			"/tmp/fortify.1971/12622d846cc3fe7b4c10359d01f0eb47/bus",
			"--filter",
			"--talk=org.freedesktop.Notifications",
			"--talk=org.freedesktop.FileManager1",
			"--talk=org.freedesktop.ScreenSaver",
			"--own=org.chromium.Chromium.*",
			"--own=org.mpris.MediaPlayer2.org.chromium.Chromium.*",
			"--own=org.mpris.MediaPlayer2.chromium.*",
			"--call=org.freedesktop.portal.*=*",
			"--broadcast=org.freedesktop.portal.*=@/org/freedesktop/portal/*",
		},
	},
	{
		"org.chromium.Chromium*", &dbus.Config{
			See:       nil,
			Talk:      []string{"org.freedesktop.Avahi", "org.freedesktop.UPower"},
			Own:       nil,
			Call:      nil,
			Broadcast: nil,
			Log:       false,
			Filter:    true,
		}, false,
		[2]string{"unix:path=/run/dbus/system_bus_socket", "/tmp/fortify.1971/12622d846cc3fe7b4c10359d01f0eb47/system_bus_socket"},
		[]string{"unix:path=/run/dbus/system_bus_socket",
			"/tmp/fortify.1971/12622d846cc3fe7b4c10359d01f0eb47/system_bus_socket",
			"--filter",
			"--talk=org.freedesktop.Avahi",
			"--talk=org.freedesktop.UPower",
		},
	},

	{
		"dev.vencord.Vesktop", &dbus.Config{
			See:       nil,
			Talk:      []string{"org.freedesktop.Notifications", "org.kde.StatusNotifierWatcher"},
			Own:       []string{"dev.vencord.Vesktop.*", "org.mpris.MediaPlayer2.dev.vencord.Vesktop.*"},
			Call:      map[string]string{"org.freedesktop.portal.*": "*"},
			Broadcast: map[string]string{"org.freedesktop.portal.*": "@/org/freedesktop/portal/*"},
			Log:       false,
			Filter:    true,
		}, false,
		[2]string{"unix:path=/run/user/1971/bus", "/tmp/fortify.1971/34c24f16a0d791d28835ededaf446033/bus"},
		[]string{
			"unix:path=/run/user/1971/bus",
			"/tmp/fortify.1971/34c24f16a0d791d28835ededaf446033/bus",
			"--filter",
			"--talk=org.freedesktop.Notifications",
			"--talk=org.kde.StatusNotifierWatcher",
			"--own=dev.vencord.Vesktop.*",
			"--own=org.mpris.MediaPlayer2.dev.vencord.Vesktop.*",
			"--call=org.freedesktop.portal.*=*",
			"--broadcast=org.freedesktop.portal.*=@/org/freedesktop/portal/*"},
	},
}

type dbusTestCase struct {
	id      string
	c       *dbus.Config
	wantErr bool
	bus     [2]string
	want    []string
}

var (
	testCasesV     []dbusTestCase
	testCasePairsV map[string][2]dbusTestCase

	testCaseOnce sync.Once
)

func testCases() []dbusTestCase {
	testCaseOnce.Do(testCaseGenerate)
	return testCasesV
}

func testCasePairs() map[string][2]dbusTestCase {
	testCaseOnce.Do(testCaseGenerate)
	return testCasePairsV
}

func injectNulls(t *[]string) {
	f := make([]string, len(*t))
	for i := range f {
		f[i] = "\x00" + (*t)[i] + "\x00"
	}
	*t = f
}

func testCaseGenerate() {
	// create null-injected test cases
	testCasesV = make([]dbusTestCase, len(samples)*2)
	for i := range samples {
		testCasesV[i] = samples[i]
		testCasesV[len(samples)+i] = samples[i]
		testCasesV[len(samples)+i].c = new(dbus.Config)
		*testCasesV[len(samples)+i].c = *samples[i].c

		// inject nulls
		fi := &testCasesV[len(samples)+i]
		fi.wantErr = true
		fi.c = &*fi.c

		injectNulls(&fi.c.See)
		injectNulls(&fi.c.Talk)
		injectNulls(&fi.c.Own)
	}

	// enumerate test case pairs
	var pc int
	for _, tc := range samples {
		if tc.id != "" {
			pc++
		}
	}
	testCasePairsV = make(map[string][2]dbusTestCase, pc)
	for i, tc := range testCasesV {
		if tc.id == "" {
			continue
		}

		// skip already enumerated system bus test
		if tc.id[len(tc.id)-1] == '*' {
			continue
		}

		ftp := [2]dbusTestCase{tc}

		// system proxy tests always place directly after its user counterpart with id ending in *
		if i+1 < len(testCasesV) && testCasesV[i+1].id[len(testCasesV[i+1].id)-1] == '*' {
			// attach system bus config
			ftp[1] = testCasesV[i+1]

			// check for misplaced/mismatching tests
			if ftp[0].wantErr != ftp[1].wantErr || ftp[0].id+"*" != ftp[1].id {
				panic("mismatching session/system pairing")
			}
		}

		k := tc.id
		if tc.wantErr {
			k = "malformed_" + k
		}
		testCasePairsV[k] = ftp
	}
}
