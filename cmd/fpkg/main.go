package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"path"
	"syscall"

	"git.gensokyo.uk/security/fortify/command"
	"git.gensokyo.uk/security/fortify/fst"
	"git.gensokyo.uk/security/fortify/internal"
	"git.gensokyo.uk/security/fortify/internal/app/instance"
	"git.gensokyo.uk/security/fortify/internal/fmsg"
	"git.gensokyo.uk/security/fortify/internal/sys"
	"git.gensokyo.uk/security/fortify/sandbox"
)

const shellPath = "/run/current-system/sw/bin/bash"

var (
	errSuccess = errors.New("success")

	std sys.State = new(sys.Std)
)

func init() {
	fmsg.Prepare("fpkg")
	if err := os.Setenv("SHELL", shellPath); err != nil {
		log.Fatalf("cannot set $SHELL: %v", err)
	}
}

func main() {
	// early init path, skips root check and duplicate PR_SET_DUMPABLE
	sandbox.TryArgv0(fmsg.Output{}, fmsg.Prepare, internal.InstallFmsg)

	if err := sandbox.SetDumpable(sandbox.SUID_DUMP_DISABLE); err != nil {
		log.Printf("cannot set SUID_DUMP_DISABLE: %s", err)
		// not fatal: this program runs as the privileged user
	}

	if os.Geteuid() == 0 {
		log.Fatal("this program must not run as root")
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop() // unreachable

	var (
		flagVerbose   bool
		flagDropShell bool
	)
	c := command.New(os.Stderr, log.Printf, "fpkg", func([]string) error {
		internal.InstallFmsg(flagVerbose)
		return nil
	}).
		Flag(&flagVerbose, "v", command.BoolFlag(false), "Print debug messages to the console").
		Flag(&flagDropShell, "s", command.BoolFlag(false), "Drop to a shell in place of next fortify action")

	c.Command("shim", command.UsageInternal, func([]string) error { instance.ShimMain(); return errSuccess })

	{
		var (
			flagDropShellActivate bool
		)
		c.NewCommand("install", "Install an application from its package", func(args []string) error {
			if len(args) != 1 {
				log.Println("invalid argument")
				return syscall.EINVAL
			}
			pkgPath := args[0]
			if !path.IsAbs(pkgPath) {
				if dir, err := os.Getwd(); err != nil {
					log.Printf("cannot get current directory: %v", err)
					return err
				} else {
					pkgPath = path.Join(dir, pkgPath)
				}
			}

			/*
				Look up paths to programs started by fpkg.
				This is done here to ease error handling as cleanup is not yet required.
			*/

			var (
				_     = lookPath("zstd")
				tar   = lookPath("tar")
				chmod = lookPath("chmod")
				rm    = lookPath("rm")
			)

			/*
				Extract package and set up for cleanup.
			*/

			var workDir string
			if p, err := os.MkdirTemp("", "fpkg.*"); err != nil {
				log.Printf("cannot create temporary directory: %v", err)
				return err
			} else {
				workDir = p
			}
			cleanup := func() {
				// should be faster than a native implementation
				mustRun(chmod, "-R", "+w", workDir)
				mustRun(rm, "-rf", workDir)
			}
			beforeRunFail.Store(&cleanup)

			mustRun(tar, "-C", workDir, "-xf", pkgPath)

			/*
				Parse bundle and app metadata, do pre-install checks.
			*/

			bundle := loadAppInfo(path.Join(workDir, "bundle.json"), cleanup)
			pathSet := pathSetByApp(bundle.ID)

			a := bundle
			if s, err := os.Stat(pathSet.metaPath); err != nil {
				if !os.IsNotExist(err) {
					cleanup()
					log.Printf("cannot access %q: %v", pathSet.metaPath, err)
					return err
				}
				// did not modify app, clean installation condition met later
			} else if s.IsDir() {
				cleanup()
				log.Printf("metadata path %q is not a file", pathSet.metaPath)
				return syscall.EBADMSG
			} else {
				a = loadAppInfo(pathSet.metaPath, cleanup)
				if a.ID != bundle.ID {
					cleanup()
					log.Printf("app %q claims to have identifier %q",
						bundle.ID, a.ID)
					return syscall.EBADE
				}
				// sec: should verify credentials
			}

			if a != bundle {
				// do not try to re-install
				if a.NixGL == bundle.NixGL &&
					a.CurrentSystem == bundle.CurrentSystem &&
					a.Launcher == bundle.Launcher &&
					a.ActivationPackage == bundle.ActivationPackage {
					cleanup()
					log.Printf("package %q is identical to local application %q",
						pkgPath, a.ID)
					return errSuccess
				}

				// AppID determines uid
				if a.AppID != bundle.AppID {
					cleanup()
					log.Printf("package %q app id %d differs from installed %d",
						pkgPath, bundle.AppID, a.AppID)
					return syscall.EBADE
				}

				// sec: should compare version string
				fmsg.Verbosef("installing application %q version %q over local %q",
					bundle.ID, bundle.Version, a.Version)
			} else {
				fmsg.Verbosef("application %q clean installation", bundle.ID)
				// sec: should install credentials
			}

			/*
				Setup steps for files owned by the target user.
			*/

			withCacheDir(ctx, "install", []string{
				// export inner bundle path in the environment
				"export BUNDLE=" + fst.Tmp + "/bundle",
				// replace inner /etc
				"mkdir -p etc",
				"chmod -R +w etc",
				"rm -rf etc",
				"cp -dRf $BUNDLE/etc etc",
				// replace inner /nix
				"mkdir -p nix",
				"chmod -R +w nix",
				"rm -rf nix",
				"cp -dRf /nix nix",
				// copy from binary cache
				"nix copy --offline --no-check-sigs --all --from file://$BUNDLE/res --to $PWD",
				// deduplicate nix store
				"nix store --offline --store $PWD optimise",
				// make cache directory world-readable for autoetc
				"chmod 0755 .",
			}, workDir, bundle, pathSet, flagDropShell, cleanup)

			if bundle.GPU {
				withCacheDir(ctx, "mesa-wrappers", []string{
					// link nixGL mesa wrappers
					"mkdir -p nix/.nixGL",
					"ln -s " + bundle.Mesa + "/bin/nixGLIntel nix/.nixGL/nixGL",
					"ln -s " + bundle.Mesa + "/bin/nixVulkanIntel nix/.nixGL/nixVulkan",
				}, workDir, bundle, pathSet, false, cleanup)
			}

			/*
				Activate home-manager generation.
			*/

			withNixDaemon(ctx, "activate", []string{
				// clean up broken links
				"mkdir -p .local/state/{nix,home-manager}",
				"chmod -R +w .local/state/{nix,home-manager}",
				"rm -rf .local/state/{nix,home-manager}",
				// run activation script
				bundle.ActivationPackage + "/activate",
			}, false, func(config *fst.Config) *fst.Config { return config },
				bundle, pathSet, flagDropShellActivate, cleanup)

			/*
				Installation complete. Write metadata to block re-installs or downgrades.
			*/

			// serialise metadata to ensure consistency
			if f, err := os.OpenFile(pathSet.metaPath+"~", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err != nil {
				cleanup()
				log.Printf("cannot create metadata file: %v", err)
				return err
			} else if err = json.NewEncoder(f).Encode(bundle); err != nil {
				cleanup()
				log.Printf("cannot write metadata: %v", err)
				return err
			} else if err = f.Close(); err != nil {
				log.Printf("cannot close metadata file: %v", err)
				// not fatal
			}

			if err := os.Rename(pathSet.metaPath+"~", pathSet.metaPath); err != nil {
				cleanup()
				log.Printf("cannot rename metadata file: %v", err)
				return err
			}

			cleanup()
			return errSuccess
		}).
			Flag(&flagDropShellActivate, "s", command.BoolFlag(false), "Drop to a shell on activation")
	}

	{
		var (
			flagDropShellNixGL bool
			flagAutoDrivers    bool
		)
		c.NewCommand("start", "Start an application", func(args []string) error {
			if len(args) < 1 {
				log.Println("invalid argument")
				return syscall.EINVAL
			}

			/*
				Parse app metadata.
			*/

			id := args[0]
			pathSet := pathSetByApp(id)
			a := loadAppInfo(pathSet.metaPath, func() {})
			if a.ID != id {
				log.Printf("app %q claims to have identifier %q", id, a.ID)
				return syscall.EBADE
			}

			/*
				Prepare nixGL.
			*/

			if a.GPU && flagAutoDrivers {
				withNixDaemon(ctx, "nix-gl", []string{
					"mkdir -p /nix/.nixGL/auto",
					"rm -rf /nix/.nixGL/auto",
					"export NIXPKGS_ALLOW_UNFREE=1",
					"nix build --impure " +
						"--out-link /nix/.nixGL/auto/opengl " +
						"--override-input nixpkgs path:/etc/nixpkgs " +
						"path:" + a.NixGL,
					"nix build --impure " +
						"--out-link /nix/.nixGL/auto/vulkan " +
						"--override-input nixpkgs path:/etc/nixpkgs " +
						"path:" + a.NixGL + "#nixVulkanNvidia",
				}, true, func(config *fst.Config) *fst.Config {
					config.Confinement.Sandbox.Filesystem = append(config.Confinement.Sandbox.Filesystem, []*fst.FilesystemConfig{
						{Src: "/etc/resolv.conf"},
						{Src: "/sys/block"},
						{Src: "/sys/bus"},
						{Src: "/sys/class"},
						{Src: "/sys/dev"},
						{Src: "/sys/devices"},
					}...)
					appendGPUFilesystem(config)
					return config
				}, a, pathSet, flagDropShellNixGL, func() {})
			}

			/*
				Create app configuration.
			*/

			argv := make([]string, 1, len(args))
			if !flagDropShell {
				argv[0] = a.Launcher
			} else {
				argv[0] = shellPath
			}
			argv = append(argv, args[1:]...)

			config := a.toFst(pathSet, argv, flagDropShell)

			/*
				Expose GPU devices.
			*/

			if a.GPU {
				config.Confinement.Sandbox.Filesystem = append(config.Confinement.Sandbox.Filesystem,
					&fst.FilesystemConfig{Src: path.Join(pathSet.nixPath, ".nixGL"), Dst: path.Join(fst.Tmp, "nixGL")})
				appendGPUFilesystem(config)
			}

			/*
				Spawn app.
			*/

			mustRunApp(ctx, config, func() {})
			return errSuccess
		}).
			Flag(&flagDropShellNixGL, "s", command.BoolFlag(false), "Drop to a shell on nixGL build").
			Flag(&flagAutoDrivers, "auto-drivers", command.BoolFlag(false), "Attempt automatic opengl driver detection")
	}

	c.MustParse(os.Args[1:], func(err error) {
		fmsg.Verbosef("command returned %v", err)
		if errors.Is(err, errSuccess) {
			fmsg.BeforeExit()
			os.Exit(0)
		}
	})
	log.Fatal("unreachable")
}
