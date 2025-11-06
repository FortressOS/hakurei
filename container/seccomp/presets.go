package seccomp

/* flatpak commit 4c3bf179e2e4a2a298cd1db1d045adaf3f564532 */

import (
	. "syscall"

	. "hakurei.app/container/std"
)

func Preset(presets FilterPreset, flags ExportFlag) (rules []NativeRule) {
	allowedPersonality := PersonaLinux
	if presets&PresetLinux32 != 0 {
		allowedPersonality = PersonaLinux32
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

	rules = make([]NativeRule, 0, l)
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

	return
}

var (
	presetCommon = []NativeRule{
		/* Block dmesg */
		{Syscall: SNR_SYSLOG, Errno: ScmpErrno(EPERM), Arg: nil},
		/* Useless old syscall */
		{Syscall: SNR_USELIB, Errno: ScmpErrno(EPERM), Arg: nil},
		/* Don't allow disabling accounting */
		{Syscall: SNR_ACCT, Errno: ScmpErrno(EPERM), Arg: nil},
		/* Don't allow reading current quota use */
		{Syscall: SNR_QUOTACTL, Errno: ScmpErrno(EPERM), Arg: nil},

		/* Don't allow access to the kernel keyring */
		{Syscall: SNR_ADD_KEY, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_KEYCTL, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_REQUEST_KEY, Errno: ScmpErrno(EPERM), Arg: nil},

		/* Scary VM/NUMA ops */
		{Syscall: SNR_MOVE_PAGES, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_MBIND, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_GET_MEMPOLICY, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SET_MEMPOLICY, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_MIGRATE_PAGES, Errno: ScmpErrno(EPERM), Arg: nil},
	}

	/* hakurei: project-specific extensions */
	presetCommonExt = []NativeRule{
		/* system calls for changing the system clock */
		{Syscall: SNR_ADJTIMEX, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_CLOCK_ADJTIME, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_CLOCK_ADJTIME64, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_CLOCK_SETTIME, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_CLOCK_SETTIME64, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETTIMEOFDAY, Errno: ScmpErrno(EPERM), Arg: nil},

		/* loading and unloading of kernel modules */
		{Syscall: SNR_DELETE_MODULE, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_FINIT_MODULE, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_INIT_MODULE, Errno: ScmpErrno(EPERM), Arg: nil},

		/* system calls for rebooting and reboot preparation */
		{Syscall: SNR_KEXEC_FILE_LOAD, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_KEXEC_LOAD, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_REBOOT, Errno: ScmpErrno(EPERM), Arg: nil},

		/* system calls for enabling/disabling swap devices */
		{Syscall: SNR_SWAPOFF, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SWAPON, Errno: ScmpErrno(EPERM), Arg: nil},
	}

	presetNamespace = []NativeRule{
		/* Don't allow subnamespace setups: */
		{Syscall: SNR_UNSHARE, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETNS, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_MOUNT, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_UMOUNT, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_UMOUNT2, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_PIVOT_ROOT, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_CHROOT, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_CLONE, Errno: ScmpErrno(EPERM),
			Arg: &ScmpArgCmp{Arg: cloneArg, Op: SCMP_CMP_MASKED_EQ, DatumA: CLONE_NEWUSER, DatumB: CLONE_NEWUSER}},

		/* seccomp can't look into clone3()'s struct clone_args to check whether
		 * the flags are OK, so we have no choice but to block clone3().
		 * Return ENOSYS so user-space will fall back to clone().
		 * (CVE-2021-41133; see also https://github.com/moby/moby/commit/9f6b562d)
		 */
		{Syscall: SNR_CLONE3, Errno: ScmpErrno(ENOSYS), Arg: nil},

		/* New mount manipulation APIs can also change our VFS. There's no
		 * legitimate reason to do these in the sandbox, so block all of them
		 * rather than thinking about which ones might be dangerous.
		 * (CVE-2021-41133) */
		{Syscall: SNR_OPEN_TREE, Errno: ScmpErrno(ENOSYS), Arg: nil},
		{Syscall: SNR_MOVE_MOUNT, Errno: ScmpErrno(ENOSYS), Arg: nil},
		{Syscall: SNR_FSOPEN, Errno: ScmpErrno(ENOSYS), Arg: nil},
		{Syscall: SNR_FSCONFIG, Errno: ScmpErrno(ENOSYS), Arg: nil},
		{Syscall: SNR_FSMOUNT, Errno: ScmpErrno(ENOSYS), Arg: nil},
		{Syscall: SNR_FSPICK, Errno: ScmpErrno(ENOSYS), Arg: nil},
		{Syscall: SNR_MOUNT_SETATTR, Errno: ScmpErrno(ENOSYS), Arg: nil},
	}

	/* hakurei: project-specific extensions */
	presetNamespaceExt = []NativeRule{
		/* changing file ownership */
		{Syscall: SNR_CHOWN, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_CHOWN32, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_FCHOWN, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_FCHOWN32, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_FCHOWNAT, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_LCHOWN, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_LCHOWN32, Errno: ScmpErrno(EPERM), Arg: nil},

		/* system calls for changing user ID and group ID credentials */
		{Syscall: SNR_SETGID, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETGID32, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETGROUPS, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETGROUPS32, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETREGID, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETREGID32, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETRESGID, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETRESGID32, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETRESUID, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETRESUID32, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETREUID, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETREUID32, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETUID, Errno: ScmpErrno(EPERM), Arg: nil},
		{Syscall: SNR_SETUID32, Errno: ScmpErrno(EPERM), Arg: nil},
	}

	presetTTY = []NativeRule{
		/* Don't allow faking input to the controlling tty (CVE-2017-5226) */
		{Syscall: SNR_IOCTL, Errno: ScmpErrno(EPERM),
			Arg: &ScmpArgCmp{Arg: 1, Op: SCMP_CMP_MASKED_EQ, DatumA: 0xFFFFFFFF, DatumB: TIOCSTI}},
		/* In the unlikely event that the controlling tty is a Linux virtual
		 * console (/dev/tty2 or similar), copy/paste operations have an effect
		 * similar to TIOCSTI (CVE-2023-28100) */
		{Syscall: SNR_IOCTL, Errno: ScmpErrno(EPERM),
			Arg: &ScmpArgCmp{Arg: 1, Op: SCMP_CMP_MASKED_EQ, DatumA: 0xFFFFFFFF, DatumB: TIOCLINUX}},
	}

	presetEmu = []NativeRule{
		/* modify_ldt is a historic source of interesting information leaks,
		 * so it's disabled as a hardening measure.
		 * However, it is required to run old 16-bit applications
		 * as well as some Wine patches, so it's allowed in multiarch. */
		{Syscall: SNR_MODIFY_LDT, Errno: ScmpErrno(EPERM), Arg: nil},
	}

	/* hakurei: project-specific extensions */
	presetEmuExt = []NativeRule{
		{Syscall: SNR_SUBPAGE_PROT, Errno: ScmpErrno(ENOSYS), Arg: nil},
		{Syscall: SNR_SWITCH_ENDIAN, Errno: ScmpErrno(ENOSYS), Arg: nil},
		{Syscall: SNR_VM86, Errno: ScmpErrno(ENOSYS), Arg: nil},
		{Syscall: SNR_VM86OLD, Errno: ScmpErrno(ENOSYS), Arg: nil},
	}
)

func presetDevel(allowedPersonality ScmpDatum) []NativeRule {
	return []NativeRule{
		/* Profiling operations; we expect these to be done by tools from outside
		 * the sandbox.  In particular perf has been the source of many CVEs. */
		{Syscall: SNR_PERF_EVENT_OPEN, Errno: ScmpErrno(EPERM), Arg: nil},
		/* Don't allow you to switch to bsd emulation or whatnot */
		{Syscall: SNR_PERSONALITY, Errno: ScmpErrno(EPERM),
			Arg: &ScmpArgCmp{Arg: 0, Op: SCMP_CMP_NE, DatumA: allowedPersonality}},

		{Syscall: SNR_PTRACE, Errno: ScmpErrno(EPERM), Arg: nil},
	}
}
