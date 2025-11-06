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
		{SNR_SYSLOG, ScmpErrno(EPERM), nil},
		/* Useless old syscall */
		{SNR_USELIB, ScmpErrno(EPERM), nil},
		/* Don't allow disabling accounting */
		{SNR_ACCT, ScmpErrno(EPERM), nil},
		/* Don't allow reading current quota use */
		{SNR_QUOTACTL, ScmpErrno(EPERM), nil},

		/* Don't allow access to the kernel keyring */
		{SNR_ADD_KEY, ScmpErrno(EPERM), nil},
		{SNR_KEYCTL, ScmpErrno(EPERM), nil},
		{SNR_REQUEST_KEY, ScmpErrno(EPERM), nil},

		/* Scary VM/NUMA ops */
		{SNR_MOVE_PAGES, ScmpErrno(EPERM), nil},
		{SNR_MBIND, ScmpErrno(EPERM), nil},
		{SNR_GET_MEMPOLICY, ScmpErrno(EPERM), nil},
		{SNR_SET_MEMPOLICY, ScmpErrno(EPERM), nil},
		{SNR_MIGRATE_PAGES, ScmpErrno(EPERM), nil},
	}

	/* hakurei: project-specific extensions */
	presetCommonExt = []NativeRule{
		/* system calls for changing the system clock */
		{SNR_ADJTIMEX, ScmpErrno(EPERM), nil},
		{SNR_CLOCK_ADJTIME, ScmpErrno(EPERM), nil},
		{SNR_CLOCK_ADJTIME64, ScmpErrno(EPERM), nil},
		{SNR_CLOCK_SETTIME, ScmpErrno(EPERM), nil},
		{SNR_CLOCK_SETTIME64, ScmpErrno(EPERM), nil},
		{SNR_SETTIMEOFDAY, ScmpErrno(EPERM), nil},

		/* loading and unloading of kernel modules */
		{SNR_DELETE_MODULE, ScmpErrno(EPERM), nil},
		{SNR_FINIT_MODULE, ScmpErrno(EPERM), nil},
		{SNR_INIT_MODULE, ScmpErrno(EPERM), nil},

		/* system calls for rebooting and reboot preparation */
		{SNR_KEXEC_FILE_LOAD, ScmpErrno(EPERM), nil},
		{SNR_KEXEC_LOAD, ScmpErrno(EPERM), nil},
		{SNR_REBOOT, ScmpErrno(EPERM), nil},

		/* system calls for enabling/disabling swap devices */
		{SNR_SWAPOFF, ScmpErrno(EPERM), nil},
		{SNR_SWAPON, ScmpErrno(EPERM), nil},
	}

	presetNamespace = []NativeRule{
		/* Don't allow subnamespace setups: */
		{SNR_UNSHARE, ScmpErrno(EPERM), nil},
		{SNR_SETNS, ScmpErrno(EPERM), nil},
		{SNR_MOUNT, ScmpErrno(EPERM), nil},
		{SNR_UMOUNT, ScmpErrno(EPERM), nil},
		{SNR_UMOUNT2, ScmpErrno(EPERM), nil},
		{SNR_PIVOT_ROOT, ScmpErrno(EPERM), nil},
		{SNR_CHROOT, ScmpErrno(EPERM), nil},
		{SNR_CLONE, ScmpErrno(EPERM),
			&ScmpArgCmp{cloneArg, SCMP_CMP_MASKED_EQ, CLONE_NEWUSER, CLONE_NEWUSER}},

		/* seccomp can't look into clone3()'s struct clone_args to check whether
		 * the flags are OK, so we have no choice but to block clone3().
		 * Return ENOSYS so user-space will fall back to clone().
		 * (CVE-2021-41133; see also https://github.com/moby/moby/commit/9f6b562d)
		 */
		{SNR_CLONE3, ScmpErrno(ENOSYS), nil},

		/* New mount manipulation APIs can also change our VFS. There's no
		 * legitimate reason to do these in the sandbox, so block all of them
		 * rather than thinking about which ones might be dangerous.
		 * (CVE-2021-41133) */
		{SNR_OPEN_TREE, ScmpErrno(ENOSYS), nil},
		{SNR_MOVE_MOUNT, ScmpErrno(ENOSYS), nil},
		{SNR_FSOPEN, ScmpErrno(ENOSYS), nil},
		{SNR_FSCONFIG, ScmpErrno(ENOSYS), nil},
		{SNR_FSMOUNT, ScmpErrno(ENOSYS), nil},
		{SNR_FSPICK, ScmpErrno(ENOSYS), nil},
		{SNR_MOUNT_SETATTR, ScmpErrno(ENOSYS), nil},
	}

	/* hakurei: project-specific extensions */
	presetNamespaceExt = []NativeRule{
		/* changing file ownership */
		{SNR_CHOWN, ScmpErrno(EPERM), nil},
		{SNR_CHOWN32, ScmpErrno(EPERM), nil},
		{SNR_FCHOWN, ScmpErrno(EPERM), nil},
		{SNR_FCHOWN32, ScmpErrno(EPERM), nil},
		{SNR_FCHOWNAT, ScmpErrno(EPERM), nil},
		{SNR_LCHOWN, ScmpErrno(EPERM), nil},
		{SNR_LCHOWN32, ScmpErrno(EPERM), nil},

		/* system calls for changing user ID and group ID credentials */
		{SNR_SETGID, ScmpErrno(EPERM), nil},
		{SNR_SETGID32, ScmpErrno(EPERM), nil},
		{SNR_SETGROUPS, ScmpErrno(EPERM), nil},
		{SNR_SETGROUPS32, ScmpErrno(EPERM), nil},
		{SNR_SETREGID, ScmpErrno(EPERM), nil},
		{SNR_SETREGID32, ScmpErrno(EPERM), nil},
		{SNR_SETRESGID, ScmpErrno(EPERM), nil},
		{SNR_SETRESGID32, ScmpErrno(EPERM), nil},
		{SNR_SETRESUID, ScmpErrno(EPERM), nil},
		{SNR_SETRESUID32, ScmpErrno(EPERM), nil},
		{SNR_SETREUID, ScmpErrno(EPERM), nil},
		{SNR_SETREUID32, ScmpErrno(EPERM), nil},
		{SNR_SETUID, ScmpErrno(EPERM), nil},
		{SNR_SETUID32, ScmpErrno(EPERM), nil},
	}

	presetTTY = []NativeRule{
		/* Don't allow faking input to the controlling tty (CVE-2017-5226) */
		{SNR_IOCTL, ScmpErrno(EPERM),
			&ScmpArgCmp{1, SCMP_CMP_MASKED_EQ, 0xFFFFFFFF, TIOCSTI}},
		/* In the unlikely event that the controlling tty is a Linux virtual
		 * console (/dev/tty2 or similar), copy/paste operations have an effect
		 * similar to TIOCSTI (CVE-2023-28100) */
		{SNR_IOCTL, ScmpErrno(EPERM),
			&ScmpArgCmp{1, SCMP_CMP_MASKED_EQ, 0xFFFFFFFF, TIOCLINUX}},
	}

	presetEmu = []NativeRule{
		/* modify_ldt is a historic source of interesting information leaks,
		 * so it's disabled as a hardening measure.
		 * However, it is required to run old 16-bit applications
		 * as well as some Wine patches, so it's allowed in multiarch. */
		{SNR_MODIFY_LDT, ScmpErrno(EPERM), nil},
	}

	/* hakurei: project-specific extensions */
	presetEmuExt = []NativeRule{
		{SNR_SUBPAGE_PROT, ScmpErrno(ENOSYS), nil},
		{SNR_SWITCH_ENDIAN, ScmpErrno(ENOSYS), nil},
		{SNR_VM86, ScmpErrno(ENOSYS), nil},
		{SNR_VM86OLD, ScmpErrno(ENOSYS), nil},
	}
)

func presetDevel(allowedPersonality ScmpDatum) []NativeRule {
	return []NativeRule{
		/* Profiling operations; we expect these to be done by tools from outside
		 * the sandbox.  In particular perf has been the source of many CVEs. */
		{SNR_PERF_EVENT_OPEN, ScmpErrno(EPERM), nil},
		/* Don't allow you to switch to bsd emulation or whatnot */
		{SNR_PERSONALITY, ScmpErrno(EPERM),
			&ScmpArgCmp{0, SCMP_CMP_NE, allowedPersonality, 0}},

		{SNR_PTRACE, ScmpErrno(EPERM), nil},
	}
}
