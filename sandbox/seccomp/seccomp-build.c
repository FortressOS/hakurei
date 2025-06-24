#ifndef _GNU_SOURCE
#define _GNU_SOURCE /* CLONE_NEWUSER */
#endif

#include "seccomp-build.h"
#include <assert.h>
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/ioctl.h>
#include <sys/personality.h>
#include <sys/socket.h>
#include <sys/syscall.h>

#if (SCMP_VER_MAJOR < 2) || (SCMP_VER_MAJOR == 2 && SCMP_VER_MINOR < 5) ||     \
    (SCMP_VER_MAJOR == 2 && SCMP_VER_MINOR == 5 && SCMP_VER_MICRO < 1)
#error This package requires libseccomp >= v2.5.1
#endif

struct hakurei_syscall_act {
  int syscall;
  int m_errno;
  struct scmp_arg_cmp *arg;
};

#define LEN(arr) (sizeof(arr) / sizeof((arr)[0]))

#define SECCOMP_RULESET_ADD(ruleset)                                           \
  do {                                                                         \
    if (opts & HAKUREI_VERBOSE)                                                      \
      hakurei_println("adding seccomp ruleset \"" #ruleset "\"");                    \
    for (int i = 0; i < LEN(ruleset); i++) {                                   \
      assert(ruleset[i].m_errno == EPERM || ruleset[i].m_errno == ENOSYS);     \
                                                                               \
      if (ruleset[i].arg)                                                      \
        *ret_p = seccomp_rule_add(ctx, SCMP_ACT_ERRNO(ruleset[i].m_errno),     \
                                  ruleset[i].syscall, 1, *ruleset[i].arg);     \
      else                                                                     \
        *ret_p = seccomp_rule_add(ctx, SCMP_ACT_ERRNO(ruleset[i].m_errno),     \
                                  ruleset[i].syscall, 0);                      \
                                                                               \
      if (*ret_p == -EFAULT) {                                                 \
        res = 4;                                                               \
        goto out;                                                              \
      } else if (*ret_p < 0) {                                                 \
        res = 5;                                                               \
        goto out;                                                              \
      }                                                                        \
    }                                                                          \
  } while (0)

int32_t hakurei_build_filter(int *ret_p, int fd, uint32_t arch, uint32_t multiarch,
                       hakurei_filter_opts opts) {
  int32_t res = 0; /* refer to resPrefix for message */
  int allow_multiarch = opts & HAKUREI_MULTIARCH;
  int allowed_personality = PER_LINUX;

  if (opts & HAKUREI_LINUX32)
    allowed_personality = PER_LINUX32;

  /* flatpak commit 4c3bf179e2e4a2a298cd1db1d045adaf3f564532 */

  struct hakurei_syscall_act deny_common[] = {
      /* Block dmesg */
      {SCMP_SYS(syslog), EPERM},
      /* Useless old syscall */
      {SCMP_SYS(uselib), EPERM},
      /* Don't allow disabling accounting */
      {SCMP_SYS(acct), EPERM},
      /* Don't allow reading current quota use */
      {SCMP_SYS(quotactl), EPERM},

      /* Don't allow access to the kernel keyring */
      {SCMP_SYS(add_key), EPERM},
      {SCMP_SYS(keyctl), EPERM},
      {SCMP_SYS(request_key), EPERM},

      /* Scary VM/NUMA ops */
      {SCMP_SYS(move_pages), EPERM},
      {SCMP_SYS(mbind), EPERM},
      {SCMP_SYS(get_mempolicy), EPERM},
      {SCMP_SYS(set_mempolicy), EPERM},
      {SCMP_SYS(migrate_pages), EPERM},
  };

  /* hakurei: project-specific extensions */
  struct hakurei_syscall_act deny_common_ext[] = {
      /* system calls for changing the system clock */
      {SCMP_SYS(adjtimex), EPERM},
      {SCMP_SYS(clock_adjtime), EPERM},
      {SCMP_SYS(clock_adjtime64), EPERM},
      {SCMP_SYS(clock_settime), EPERM},
      {SCMP_SYS(clock_settime64), EPERM},
      {SCMP_SYS(settimeofday), EPERM},

      /* loading and unloading of kernel modules */
      {SCMP_SYS(delete_module), EPERM},
      {SCMP_SYS(finit_module), EPERM},
      {SCMP_SYS(init_module), EPERM},

      /* system calls for rebooting and reboot preparation */
      {SCMP_SYS(kexec_file_load), EPERM},
      {SCMP_SYS(kexec_load), EPERM},
      {SCMP_SYS(reboot), EPERM},

      /* system calls for enabling/disabling swap devices */
      {SCMP_SYS(swapoff), EPERM},
      {SCMP_SYS(swapon), EPERM},
  };

  struct hakurei_syscall_act deny_ns[] = {
      /* Don't allow subnamespace setups: */
      {SCMP_SYS(unshare), EPERM},
      {SCMP_SYS(setns), EPERM},
      {SCMP_SYS(mount), EPERM},
      {SCMP_SYS(umount), EPERM},
      {SCMP_SYS(umount2), EPERM},
      {SCMP_SYS(pivot_root), EPERM},
      {SCMP_SYS(chroot), EPERM},
#if defined(__s390__) || defined(__s390x__) || defined(__CRIS__)
      /* Architectures with CONFIG_CLONE_BACKWARDS2: the child stack
       * and flags arguments are reversed so the flags come second */
      {SCMP_SYS(clone), EPERM,
       &SCMP_A1(SCMP_CMP_MASKED_EQ, CLONE_NEWUSER, CLONE_NEWUSER)},
#else
      /* Normally the flags come first */
      {SCMP_SYS(clone), EPERM,
       &SCMP_A0(SCMP_CMP_MASKED_EQ, CLONE_NEWUSER, CLONE_NEWUSER)},
#endif

      /* seccomp can't look into clone3()'s struct clone_args to check whether
       * the flags are OK, so we have no choice but to block clone3().
       * Return ENOSYS so user-space will fall back to clone().
       * (CVE-2021-41133; see also https://github.com/moby/moby/commit/9f6b562d)
       */
      {SCMP_SYS(clone3), ENOSYS},

      /* New mount manipulation APIs can also change our VFS. There's no
       * legitimate reason to do these in the sandbox, so block all of them
       * rather than thinking about which ones might be dangerous.
       * (CVE-2021-41133) */
      {SCMP_SYS(open_tree), ENOSYS},
      {SCMP_SYS(move_mount), ENOSYS},
      {SCMP_SYS(fsopen), ENOSYS},
      {SCMP_SYS(fsconfig), ENOSYS},
      {SCMP_SYS(fsmount), ENOSYS},
      {SCMP_SYS(fspick), ENOSYS},
      {SCMP_SYS(mount_setattr), ENOSYS},
  };

  /* hakurei: project-specific extensions */
  struct hakurei_syscall_act deny_ns_ext[] = {
      /* changing file ownership */
      {SCMP_SYS(chown), EPERM},
      {SCMP_SYS(chown32), EPERM},
      {SCMP_SYS(fchown), EPERM},
      {SCMP_SYS(fchown32), EPERM},
      {SCMP_SYS(fchownat), EPERM},
      {SCMP_SYS(lchown), EPERM},
      {SCMP_SYS(lchown32), EPERM},

      /* system calls for changing user ID and group ID credentials */
      {SCMP_SYS(setgid), EPERM},
      {SCMP_SYS(setgid32), EPERM},
      {SCMP_SYS(setgroups), EPERM},
      {SCMP_SYS(setgroups32), EPERM},
      {SCMP_SYS(setregid), EPERM},
      {SCMP_SYS(setregid32), EPERM},
      {SCMP_SYS(setresgid), EPERM},
      {SCMP_SYS(setresgid32), EPERM},
      {SCMP_SYS(setresuid), EPERM},
      {SCMP_SYS(setresuid32), EPERM},
      {SCMP_SYS(setreuid), EPERM},
      {SCMP_SYS(setreuid32), EPERM},
      {SCMP_SYS(setuid), EPERM},
      {SCMP_SYS(setuid32), EPERM},
  };

  struct hakurei_syscall_act deny_tty[] = {
      /* Don't allow faking input to the controlling tty (CVE-2017-5226) */
      {SCMP_SYS(ioctl), EPERM,
       &SCMP_A1(SCMP_CMP_MASKED_EQ, 0xFFFFFFFFu, (int)TIOCSTI)},
      /* In the unlikely event that the controlling tty is a Linux virtual
       * console (/dev/tty2 or similar), copy/paste operations have an effect
       * similar to TIOCSTI (CVE-2023-28100) */
      {SCMP_SYS(ioctl), EPERM,
       &SCMP_A1(SCMP_CMP_MASKED_EQ, 0xFFFFFFFFu, (int)TIOCLINUX)},
  };

  struct hakurei_syscall_act deny_devel[] = {
      /* Profiling operations; we expect these to be done by tools from outside
       * the sandbox.  In particular perf has been the source of many CVEs. */
      {SCMP_SYS(perf_event_open), EPERM},
      /* Don't allow you to switch to bsd emulation or whatnot */
      {SCMP_SYS(personality), EPERM,
       &SCMP_A0(SCMP_CMP_NE, allowed_personality)},

      {SCMP_SYS(ptrace), EPERM}};

  struct hakurei_syscall_act deny_emu[] = {
      /* modify_ldt is a historic source of interesting information leaks,
       * so it's disabled as a hardening measure.
       * However, it is required to run old 16-bit applications
       * as well as some Wine patches, so it's allowed in multiarch. */
      {SCMP_SYS(modify_ldt), EPERM},
  };

  /* hakurei: project-specific extensions */
  struct hakurei_syscall_act deny_emu_ext[] = {
      {SCMP_SYS(subpage_prot), ENOSYS},
      {SCMP_SYS(switch_endian), ENOSYS},
      {SCMP_SYS(vm86), ENOSYS},
      {SCMP_SYS(vm86old), ENOSYS},
  };

  /* Blocklist all but unix, inet, inet6 and netlink */
  struct {
    int family;
    hakurei_filter_opts flags_mask;
  } socket_family_allowlist[] = {
      /* NOTE: Keep in numerical order */
      {AF_UNSPEC, 0},
      {AF_LOCAL, 0},
      {AF_INET, 0},
      {AF_INET6, 0},
      {AF_NETLINK, 0},
      {AF_CAN, HAKUREI_CAN},
      {AF_BLUETOOTH, HAKUREI_BLUETOOTH},
  };

  scmp_filter_ctx ctx = seccomp_init(SCMP_ACT_ALLOW);
  if (ctx == NULL) {
    res = 1;
    goto out;
  } else
    errno = 0;

  /* We only really need to handle arches on multiarch systems.
   * If only one arch is supported the default is fine */
  if (arch != 0) {
    /* This *adds* the target arch, instead of replacing the
     * native one. This is not ideal, because we'd like to only
     * allow the target arch, but we can't really disallow the
     * native arch at this point, because then bubblewrap
     * couldn't continue running. */
    *ret_p = seccomp_arch_add(ctx, arch);
    if (*ret_p < 0 && *ret_p != -EEXIST) {
      res = 2;
      goto out;
    }

    if (allow_multiarch && multiarch != 0) {
      *ret_p = seccomp_arch_add(ctx, multiarch);
      if (*ret_p < 0 && *ret_p != -EEXIST) {
        res = 3;
        goto out;
      }
    }
  }

  SECCOMP_RULESET_ADD(deny_common);
  if (opts & HAKUREI_DENY_NS)
    SECCOMP_RULESET_ADD(deny_ns);
  if (opts & HAKUREI_DENY_TTY)
    SECCOMP_RULESET_ADD(deny_tty);
  if (opts & HAKUREI_DENY_DEVEL)
    SECCOMP_RULESET_ADD(deny_devel);
  if (!allow_multiarch)
    SECCOMP_RULESET_ADD(deny_emu);
  if (opts & HAKUREI_EXT) {
    SECCOMP_RULESET_ADD(deny_common_ext);
    if (opts & HAKUREI_DENY_NS)
      SECCOMP_RULESET_ADD(deny_ns_ext);
    if (!allow_multiarch)
      SECCOMP_RULESET_ADD(deny_emu_ext);
  }

  /* Socket filtering doesn't work on e.g. i386, so ignore failures here
   * However, we need to user seccomp_rule_add_exact to avoid libseccomp doing
   * something else: https://github.com/seccomp/libseccomp/issues/8 */
  int last_allowed_family = -1;
  for (int i = 0; i < LEN(socket_family_allowlist); i++) {
    if (socket_family_allowlist[i].flags_mask != 0 &&
        (socket_family_allowlist[i].flags_mask & opts) !=
            socket_family_allowlist[i].flags_mask)
      continue;

    for (int disallowed = last_allowed_family + 1;
         disallowed < socket_family_allowlist[i].family; disallowed++) {
      /* Blocklist the in-between valid families */
      seccomp_rule_add_exact(ctx, SCMP_ACT_ERRNO(EAFNOSUPPORT),
                             SCMP_SYS(socket), 1,
                             SCMP_A0(SCMP_CMP_EQ, disallowed));
    }
    last_allowed_family = socket_family_allowlist[i].family;
  }
  /* Blocklist the rest */
  seccomp_rule_add_exact(ctx, SCMP_ACT_ERRNO(EAFNOSUPPORT), SCMP_SYS(socket), 1,
                         SCMP_A0(SCMP_CMP_GE, last_allowed_family + 1));

  if (fd < 0) {
    *ret_p = seccomp_load(ctx);
    if (*ret_p != 0) {
      res = 7;
      goto out;
    }
  } else {
    *ret_p = seccomp_export_bpf(ctx, fd);
    if (*ret_p != 0) {
      res = 6;
      goto out;
    }
  }

out:
  if (ctx)
    seccomp_release(ctx);

  return res;
}
