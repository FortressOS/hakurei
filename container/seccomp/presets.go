package seccomp

/* flatpak commit 4c3bf179e2e4a2a298cd1db1d045adaf3f564532 */

import (
	. "syscall"

	"hakurei.app/container/bits"
)

func Preset(presets bits.FilterPreset, flags ExportFlag) (rules []NativeRule) {
	allowedPersonality := PER_LINUX
	if presets&bits.PresetLinux32 != 0 {
		allowedPersonality = PER_LINUX32
	}
	presetDevelFinal := presetDevel(ScmpDatum(allowedPersonality))

	l := len(presetCommon)
	if presets&bits.PresetDenyNS != 0 {
		l += len(presetNamespace)
	}
	if presets&bits.PresetDenyTTY != 0 {
		l += len(presetTTY)
	}
	if presets&bits.PresetDenyDevel != 0 {
		l += len(presetDevelFinal)
	}
	if flags&AllowMultiarch == 0 {
		l += len(presetEmu)
	}
	if presets&bits.PresetExt != 0 {
		l += len(presetCommonExt)
		if presets&bits.PresetDenyNS != 0 {
			l += len(presetNamespaceExt)
		}
		if flags&AllowMultiarch == 0 {
			l += len(presetEmuExt)
		}
	}

	rules = make([]NativeRule, 0, l)
	rules = append(rules, presetCommon...)
	if presets&bits.PresetDenyNS != 0 {
		rules = append(rules, presetNamespace...)
	}
	if presets&bits.PresetDenyTTY != 0 {
		rules = append(rules, presetTTY...)
	}
	if presets&bits.PresetDenyDevel != 0 {
		rules = append(rules, presetDevelFinal...)
	}
	if flags&AllowMultiarch == 0 {
		rules = append(rules, presetEmu...)
	}
	if presets&bits.PresetExt != 0 {
		rules = append(rules, presetCommonExt...)
		if presets&bits.PresetDenyNS != 0 {
			rules = append(rules, presetNamespaceExt...)
		}
		if flags&AllowMultiarch == 0 {
			rules = append(rules, presetEmuExt...)
		}
	}

	return
}

var (
	presetCommon = []NativeRule{
		/* Block dmesg */
		{ScmpSyscall(SYS_SYSLOG), ScmpErrno(EPERM), nil},
		/* Useless old syscall */
		{ScmpSyscall(SYS_USELIB), ScmpErrno(EPERM), nil},
		/* Don't allow disabling accounting */
		{ScmpSyscall(SYS_ACCT), ScmpErrno(EPERM), nil},
		/* Don't allow reading current quota use */
		{ScmpSyscall(SYS_QUOTACTL), ScmpErrno(EPERM), nil},

		/* Don't allow access to the kernel keyring */
		{ScmpSyscall(SYS_ADD_KEY), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_KEYCTL), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_REQUEST_KEY), ScmpErrno(EPERM), nil},

		/* Scary VM/NUMA ops */
		{ScmpSyscall(SYS_MOVE_PAGES), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_MBIND), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_GET_MEMPOLICY), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SET_MEMPOLICY), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_MIGRATE_PAGES), ScmpErrno(EPERM), nil},
	}

	/* hakurei: project-specific extensions */
	presetCommonExt = []NativeRule{
		/* system calls for changing the system clock */
		{ScmpSyscall(SYS_ADJTIMEX), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_CLOCK_ADJTIME), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_CLOCK_ADJTIME64), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_CLOCK_SETTIME), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_CLOCK_SETTIME64), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETTIMEOFDAY), ScmpErrno(EPERM), nil},

		/* loading and unloading of kernel modules */
		{ScmpSyscall(SYS_DELETE_MODULE), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_FINIT_MODULE), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_INIT_MODULE), ScmpErrno(EPERM), nil},

		/* system calls for rebooting and reboot preparation */
		{ScmpSyscall(SYS_KEXEC_FILE_LOAD), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_KEXEC_LOAD), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_REBOOT), ScmpErrno(EPERM), nil},

		/* system calls for enabling/disabling swap devices */
		{ScmpSyscall(SYS_SWAPOFF), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SWAPON), ScmpErrno(EPERM), nil},
	}

	presetNamespace = []NativeRule{
		/* Don't allow subnamespace setups: */
		{ScmpSyscall(SYS_UNSHARE), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETNS), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_MOUNT), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_UMOUNT), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_UMOUNT2), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_PIVOT_ROOT), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_CHROOT), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_CLONE), ScmpErrno(EPERM),
			&ScmpArgCmp{cloneArg, SCMP_CMP_MASKED_EQ, CLONE_NEWUSER, CLONE_NEWUSER}},

		/* seccomp can't look into clone3()'s struct clone_args to check whether
		 * the flags are OK, so we have no choice but to block clone3().
		 * Return ENOSYS so user-space will fall back to clone().
		 * (CVE-2021-41133; see also https://github.com/moby/moby/commit/9f6b562d)
		 */
		{ScmpSyscall(SYS_CLONE3), ScmpErrno(ENOSYS), nil},

		/* New mount manipulation APIs can also change our VFS. There's no
		 * legitimate reason to do these in the sandbox, so block all of them
		 * rather than thinking about which ones might be dangerous.
		 * (CVE-2021-41133) */
		{ScmpSyscall(SYS_OPEN_TREE), ScmpErrno(ENOSYS), nil},
		{ScmpSyscall(SYS_MOVE_MOUNT), ScmpErrno(ENOSYS), nil},
		{ScmpSyscall(SYS_FSOPEN), ScmpErrno(ENOSYS), nil},
		{ScmpSyscall(SYS_FSCONFIG), ScmpErrno(ENOSYS), nil},
		{ScmpSyscall(SYS_FSMOUNT), ScmpErrno(ENOSYS), nil},
		{ScmpSyscall(SYS_FSPICK), ScmpErrno(ENOSYS), nil},
		{ScmpSyscall(SYS_MOUNT_SETATTR), ScmpErrno(ENOSYS), nil},
	}

	/* hakurei: project-specific extensions */
	presetNamespaceExt = []NativeRule{
		/* changing file ownership */
		{ScmpSyscall(SYS_CHOWN), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_CHOWN32), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_FCHOWN), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_FCHOWN32), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_FCHOWNAT), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_LCHOWN), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_LCHOWN32), ScmpErrno(EPERM), nil},

		/* system calls for changing user ID and group ID credentials */
		{ScmpSyscall(SYS_SETGID), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETGID32), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETGROUPS), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETGROUPS32), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETREGID), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETREGID32), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETRESGID), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETRESGID32), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETRESUID), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETRESUID32), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETREUID), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETREUID32), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETUID), ScmpErrno(EPERM), nil},
		{ScmpSyscall(SYS_SETUID32), ScmpErrno(EPERM), nil},
	}

	presetTTY = []NativeRule{
		/* Don't allow faking input to the controlling tty (CVE-2017-5226) */
		{ScmpSyscall(SYS_IOCTL), ScmpErrno(EPERM),
			&ScmpArgCmp{1, SCMP_CMP_MASKED_EQ, 0xFFFFFFFF, TIOCSTI}},
		/* In the unlikely event that the controlling tty is a Linux virtual
		 * console (/dev/tty2 or similar), copy/paste operations have an effect
		 * similar to TIOCSTI (CVE-2023-28100) */
		{ScmpSyscall(SYS_IOCTL), ScmpErrno(EPERM),
			&ScmpArgCmp{1, SCMP_CMP_MASKED_EQ, 0xFFFFFFFF, TIOCLINUX}},
	}

	presetEmu = []NativeRule{
		/* modify_ldt is a historic source of interesting information leaks,
		 * so it's disabled as a hardening measure.
		 * However, it is required to run old 16-bit applications
		 * as well as some Wine patches, so it's allowed in multiarch. */
		{ScmpSyscall(SYS_MODIFY_LDT), ScmpErrno(EPERM), nil},
	}

	/* hakurei: project-specific extensions */
	presetEmuExt = []NativeRule{
		{ScmpSyscall(SYS_SUBPAGE_PROT), ScmpErrno(ENOSYS), nil},
		{ScmpSyscall(SYS_SWITCH_ENDIAN), ScmpErrno(ENOSYS), nil},
		{ScmpSyscall(SYS_VM86), ScmpErrno(ENOSYS), nil},
		{ScmpSyscall(SYS_VM86OLD), ScmpErrno(ENOSYS), nil},
	}
)

func presetDevel(allowedPersonality ScmpDatum) []NativeRule {
	return []NativeRule{
		/* Profiling operations; we expect these to be done by tools from outside
		 * the sandbox.  In particular perf has been the source of many CVEs. */
		{ScmpSyscall(SYS_PERF_EVENT_OPEN), ScmpErrno(EPERM), nil},
		/* Don't allow you to switch to bsd emulation or whatnot */
		{ScmpSyscall(SYS_PERSONALITY), ScmpErrno(EPERM),
			&ScmpArgCmp{0, SCMP_CMP_NE, allowedPersonality, 0}},

		{ScmpSyscall(SYS_PTRACE), ScmpErrno(EPERM), nil},
	}
}
