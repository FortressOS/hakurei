package seccomp

/* flatpak commit 4c3bf179e2e4a2a298cd1db1d045adaf3f564532 */

import "C"
import (
	. "syscall"
)

type FilterPreset int

const (
	// PresetExt are project-specific extensions.
	PresetExt FilterPreset = 1 << iota
	// PresetDenyNS denies namespace setup syscalls.
	PresetDenyNS
	// PresetDenyTTY denies faking input.
	PresetDenyTTY
	// PresetDenyDevel denies development-related syscalls.
	PresetDenyDevel
	// PresetLinux32 sets PER_LINUX32.
	PresetLinux32
)

func preparePreset(fd int, presets FilterPreset, flags PrepareFlag) error {
	allowedPersonality := PER_LINUX
	if presets&PresetLinux32 != 0 {
		allowedPersonality = PER_LINUX32
	}
	presetDevelFinal := presetDevel(ScmpDatum(allowedPersonality))

	l := len(presetCommon)
	if presets&PresetDenyNS != 0 {
		l += len(presetNamespace)
	}
	if presets&PresetDenyTTY != 0 {
		l += len(presetTTY)
	}
	if presets&PresetDenyDevel != 0 {
		l += len(presetDevelFinal)
	}
	if flags&AllowMultiarch == 0 {
		l += len(presetEmu)
	}
	if presets&PresetExt != 0 {
		l += len(presetCommonExt)
		if presets&PresetDenyNS != 0 {
			l += len(presetNamespaceExt)
		}
		if flags&AllowMultiarch == 0 {
			l += len(presetEmuExt)
		}
	}

	rules := make([]NativeRule, 0, l)
	rules = append(rules, presetCommon...)
	if presets&PresetDenyNS != 0 {
		rules = append(rules, presetNamespace...)
	}
	if presets&PresetDenyTTY != 0 {
		rules = append(rules, presetTTY...)
	}
	if presets&PresetDenyDevel != 0 {
		rules = append(rules, presetDevelFinal...)
	}
	if flags&AllowMultiarch == 0 {
		rules = append(rules, presetEmu...)
	}
	if presets&PresetExt != 0 {
		rules = append(rules, presetCommonExt...)
		if presets&PresetDenyNS != 0 {
			rules = append(rules, presetNamespaceExt...)
		}
		if flags&AllowMultiarch == 0 {
			rules = append(rules, presetEmuExt...)
		}
	}

	return Prepare(fd, rules, flags)
}

var (
	presetCommon = []NativeRule{
		/* Block dmesg */
		{C.int(SYS_SYSLOG), C.int(EPERM), nil},
		/* Useless old syscall */
		{C.int(SYS_USELIB), C.int(EPERM), nil},
		/* Don't allow disabling accounting */
		{C.int(SYS_ACCT), C.int(EPERM), nil},
		/* Don't allow reading current quota use */
		{C.int(SYS_QUOTACTL), C.int(EPERM), nil},

		/* Don't allow access to the kernel keyring */
		{C.int(SYS_ADD_KEY), C.int(EPERM), nil},
		{C.int(SYS_KEYCTL), C.int(EPERM), nil},
		{C.int(SYS_REQUEST_KEY), C.int(EPERM), nil},

		/* Scary VM/NUMA ops */
		{C.int(SYS_MOVE_PAGES), C.int(EPERM), nil},
		{C.int(SYS_MBIND), C.int(EPERM), nil},
		{C.int(SYS_GET_MEMPOLICY), C.int(EPERM), nil},
		{C.int(SYS_SET_MEMPOLICY), C.int(EPERM), nil},
		{C.int(SYS_MIGRATE_PAGES), C.int(EPERM), nil},
	}

	/* hakurei: project-specific extensions */
	presetCommonExt = []NativeRule{
		/* system calls for changing the system clock */
		{C.int(SYS_ADJTIMEX), C.int(EPERM), nil},
		{C.int(SYS_CLOCK_ADJTIME), C.int(EPERM), nil},
		{C.int(SYS_CLOCK_ADJTIME64), C.int(EPERM), nil},
		{C.int(SYS_CLOCK_SETTIME), C.int(EPERM), nil},
		{C.int(SYS_CLOCK_SETTIME64), C.int(EPERM), nil},
		{C.int(SYS_SETTIMEOFDAY), C.int(EPERM), nil},

		/* loading and unloading of kernel modules */
		{C.int(SYS_DELETE_MODULE), C.int(EPERM), nil},
		{C.int(SYS_FINIT_MODULE), C.int(EPERM), nil},
		{C.int(SYS_INIT_MODULE), C.int(EPERM), nil},

		/* system calls for rebooting and reboot preparation */
		{C.int(SYS_KEXEC_FILE_LOAD), C.int(EPERM), nil},
		{C.int(SYS_KEXEC_LOAD), C.int(EPERM), nil},
		{C.int(SYS_REBOOT), C.int(EPERM), nil},

		/* system calls for enabling/disabling swap devices */
		{C.int(SYS_SWAPOFF), C.int(EPERM), nil},
		{C.int(SYS_SWAPON), C.int(EPERM), nil},
	}

	presetNamespace = []NativeRule{
		/* Don't allow subnamespace setups: */
		{C.int(SYS_UNSHARE), C.int(EPERM), nil},
		{C.int(SYS_SETNS), C.int(EPERM), nil},
		{C.int(SYS_MOUNT), C.int(EPERM), nil},
		{C.int(SYS_UMOUNT), C.int(EPERM), nil},
		{C.int(SYS_UMOUNT2), C.int(EPERM), nil},
		{C.int(SYS_PIVOT_ROOT), C.int(EPERM), nil},
		{C.int(SYS_CHROOT), C.int(EPERM), nil},
		{C.int(SYS_CLONE), C.int(EPERM),
			&ScmpArgCmp{cloneArg, SCMP_CMP_MASKED_EQ, CLONE_NEWUSER, CLONE_NEWUSER}},

		/* seccomp can't look into clone3()'s struct clone_args to check whether
		 * the flags are OK, so we have no choice but to block clone3().
		 * Return ENOSYS so user-space will fall back to clone().
		 * (CVE-2021-41133; see also https://github.com/moby/moby/commit/9f6b562d)
		 */
		{C.int(SYS_CLONE3), C.int(ENOSYS), nil},

		/* New mount manipulation APIs can also change our VFS. There's no
		 * legitimate reason to do these in the sandbox, so block all of them
		 * rather than thinking about which ones might be dangerous.
		 * (CVE-2021-41133) */
		{C.int(SYS_OPEN_TREE), C.int(ENOSYS), nil},
		{C.int(SYS_MOVE_MOUNT), C.int(ENOSYS), nil},
		{C.int(SYS_FSOPEN), C.int(ENOSYS), nil},
		{C.int(SYS_FSCONFIG), C.int(ENOSYS), nil},
		{C.int(SYS_FSMOUNT), C.int(ENOSYS), nil},
		{C.int(SYS_FSPICK), C.int(ENOSYS), nil},
		{C.int(SYS_MOUNT_SETATTR), C.int(ENOSYS), nil},
	}

	/* hakurei: project-specific extensions */
	presetNamespaceExt = []NativeRule{
		/* changing file ownership */
		{C.int(SYS_CHOWN), C.int(EPERM), nil},
		{C.int(SYS_CHOWN32), C.int(EPERM), nil},
		{C.int(SYS_FCHOWN), C.int(EPERM), nil},
		{C.int(SYS_FCHOWN32), C.int(EPERM), nil},
		{C.int(SYS_FCHOWNAT), C.int(EPERM), nil},
		{C.int(SYS_LCHOWN), C.int(EPERM), nil},
		{C.int(SYS_LCHOWN32), C.int(EPERM), nil},

		/* system calls for changing user ID and group ID credentials */
		{C.int(SYS_SETGID), C.int(EPERM), nil},
		{C.int(SYS_SETGID32), C.int(EPERM), nil},
		{C.int(SYS_SETGROUPS), C.int(EPERM), nil},
		{C.int(SYS_SETGROUPS32), C.int(EPERM), nil},
		{C.int(SYS_SETREGID), C.int(EPERM), nil},
		{C.int(SYS_SETREGID32), C.int(EPERM), nil},
		{C.int(SYS_SETRESGID), C.int(EPERM), nil},
		{C.int(SYS_SETRESGID32), C.int(EPERM), nil},
		{C.int(SYS_SETRESUID), C.int(EPERM), nil},
		{C.int(SYS_SETRESUID32), C.int(EPERM), nil},
		{C.int(SYS_SETREUID), C.int(EPERM), nil},
		{C.int(SYS_SETREUID32), C.int(EPERM), nil},
		{C.int(SYS_SETUID), C.int(EPERM), nil},
		{C.int(SYS_SETUID32), C.int(EPERM), nil},
	}

	presetTTY = []NativeRule{
		/* Don't allow faking input to the controlling tty (CVE-2017-5226) */
		{C.int(SYS_IOCTL), C.int(EPERM),
			&ScmpArgCmp{1, SCMP_CMP_MASKED_EQ, 0xFFFFFFFF, TIOCSTI}},
		/* In the unlikely event that the controlling tty is a Linux virtual
		 * console (/dev/tty2 or similar), copy/paste operations have an effect
		 * similar to TIOCSTI (CVE-2023-28100) */
		{C.int(SYS_IOCTL), C.int(EPERM),
			&ScmpArgCmp{1, SCMP_CMP_MASKED_EQ, 0xFFFFFFFF, TIOCLINUX}},
	}

	presetEmu = []NativeRule{
		/* modify_ldt is a historic source of interesting information leaks,
		 * so it's disabled as a hardening measure.
		 * However, it is required to run old 16-bit applications
		 * as well as some Wine patches, so it's allowed in multiarch. */
		{C.int(SYS_MODIFY_LDT), C.int(EPERM), nil},
	}

	/* hakurei: project-specific extensions */
	presetEmuExt = []NativeRule{
		{C.int(SYS_SUBPAGE_PROT), C.int(ENOSYS), nil},
		{C.int(SYS_SWITCH_ENDIAN), C.int(ENOSYS), nil},
		{C.int(SYS_VM86), C.int(ENOSYS), nil},
		{C.int(SYS_VM86OLD), C.int(ENOSYS), nil},
	}
)

func presetDevel(allowedPersonality ScmpDatum) []NativeRule {
	return []NativeRule{
		/* Profiling operations; we expect these to be done by tools from outside
		 * the sandbox.  In particular perf has been the source of many CVEs. */
		{C.int(SYS_PERF_EVENT_OPEN), C.int(EPERM), nil},
		/* Don't allow you to switch to bsd emulation or whatnot */
		{C.int(SYS_PERSONALITY), C.int(EPERM),
			&ScmpArgCmp{0, SCMP_CMP_NE, allowedPersonality, 0}},

		{C.int(SYS_PTRACE), C.int(EPERM), nil},
	}
}
