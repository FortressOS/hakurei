#ifndef _GNU_SOURCE
#define _GNU_SOURCE // CLONE_NEWUSER
#endif

#include "export.h"
#include <stdlib.h>
#include <stdio.h>
#include <assert.h>
#include <errno.h>
#include <sys/syscall.h>
#include <sys/socket.h>
#include <sys/ioctl.h>
#include <sys/personality.h>
#include <sched.h>

#if (SCMP_VER_MAJOR < 2) || \
    (SCMP_VER_MAJOR == 2 && SCMP_VER_MINOR < 5) || \
    (SCMP_VER_MAJOR == 2 && SCMP_VER_MINOR == 5 && SCMP_VER_MICRO < 1)
#error This package requires libseccomp >= v2.5.1
#endif

struct f_syscall_act {
  int                  syscall;
  int                  m_errno;
  struct scmp_arg_cmp *arg;
};

#define LEN(arr) (sizeof(arr) / sizeof((arr)[0]))

#define SECCOMP_RULESET_ADD(ruleset) do {                                                                      \
  F_println("adding seccomp ruleset \"" #ruleset "\""); \
  for (int i = 0; i < LEN(ruleset); i++) {                                                                     \
    assert(ruleset[i].m_errno == EPERM || ruleset[i].m_errno == ENOSYS);                                       \
                                                                                                               \
    if (ruleset[i].arg)                                                                                        \
      ret = seccomp_rule_add(ctx, SCMP_ACT_ERRNO(ruleset[i].m_errno), ruleset[i].syscall, 1, *ruleset[i].arg); \
    else                                                                                                       \
      ret = seccomp_rule_add(ctx, SCMP_ACT_ERRNO(ruleset[i].m_errno), ruleset[i].syscall, 0);                  \
                                                                                                               \
    if (ret == -EFAULT) {                                                                                      \
      res = 4;                                                                                                 \
      goto out;                                                                                                \
    } else if (ret < 0) {                                                                                      \
      res = 5;                                                                                                 \
      errno = -ret;                                                                                            \
      goto out;                                                                                                \
    }                                                                                                          \
  }                                                                                                            \
} while (0)


int f_tmpfile_fd() {
  FILE *f = tmpfile();
  if (f == NULL)
    return -1;
  return fileno(f);
}

int32_t f_export_bpf(int fd, uint32_t arch, uint32_t multiarch, f_syscall_opts opts) {
  int32_t res = 0; // refer to resErr for meaning
  int allow_multiarch = opts & F_MULTIARCH;
  int allowed_personality = PER_LINUX;

  if (opts & F_LINUX32)
    allowed_personality = PER_LINUX32;

  // flatpak commit 4c3bf179e2e4a2a298cd1db1d045adaf3f564532

  struct f_syscall_act deny_common[] = {
    // Block dmesg
    {SCMP_SYS(syslog), EPERM},
    // Useless old syscall
    {SCMP_SYS(uselib), EPERM},
    // Don't allow disabling accounting
    {SCMP_SYS(acct), EPERM},
    // Don't allow reading current quota use
    {SCMP_SYS(quotactl), EPERM},

    // Don't allow access to the kernel keyring
    {SCMP_SYS(add_key), EPERM},
    {SCMP_SYS(keyctl), EPERM},
    {SCMP_SYS(request_key), EPERM},

    // Scary VM/NUMA ops
    {SCMP_SYS(move_pages), EPERM},
    {SCMP_SYS(mbind), EPERM},
    {SCMP_SYS(get_mempolicy), EPERM},
    {SCMP_SYS(set_mempolicy), EPERM},
    {SCMP_SYS(migrate_pages), EPERM},
  };

  struct f_syscall_act deny_ns[] = {
    // Don't allow subnamespace setups:
    {SCMP_SYS(unshare), EPERM},
    {SCMP_SYS(setns), EPERM},
    {SCMP_SYS(mount), EPERM},
    {SCMP_SYS(umount), EPERM},
    {SCMP_SYS(umount2), EPERM},
    {SCMP_SYS(pivot_root), EPERM},
    {SCMP_SYS(chroot), EPERM},
#if defined(__s390__) || defined(__s390x__) || defined(__CRIS__)
    // Architectures with CONFIG_CLONE_BACKWARDS2: the child stack
    // and flags arguments are reversed so the flags come second
    {SCMP_SYS(clone), EPERM, &SCMP_A1(SCMP_CMP_MASKED_EQ, CLONE_NEWUSER, CLONE_NEWUSER)},
#else
    // Normally the flags come first
    {SCMP_SYS(clone), EPERM, &SCMP_A0(SCMP_CMP_MASKED_EQ, CLONE_NEWUSER, CLONE_NEWUSER)},
#endif

    // seccomp can't look into clone3()'s struct clone_args to check whether
    // the flags are OK, so we have no choice but to block clone3().
    // Return ENOSYS so user-space will fall back to clone().
    // (CVE-2021-41133; see also https://github.com/moby/moby/commit/9f6b562d)
    {SCMP_SYS(clone3), ENOSYS},

    // New mount manipulation APIs can also change our VFS. There's no
    // legitimate reason to do these in the sandbox, so block all of them
    // rather than thinking about which ones might be dangerous.
    // (CVE-2021-41133)
    {SCMP_SYS(open_tree), ENOSYS},
    {SCMP_SYS(move_mount), ENOSYS},
    {SCMP_SYS(fsopen), ENOSYS},
    {SCMP_SYS(fsconfig), ENOSYS},
    {SCMP_SYS(fsmount), ENOSYS},
    {SCMP_SYS(fspick), ENOSYS},
    {SCMP_SYS(mount_setattr), ENOSYS},
  };

  struct f_syscall_act deny_tty[] = {
    // Don't allow faking input to the controlling tty (CVE-2017-5226)
    {SCMP_SYS(ioctl), EPERM, &SCMP_A1(SCMP_CMP_MASKED_EQ, 0xFFFFFFFFu, (int)TIOCSTI)},
    // In the unlikely event that the controlling tty is a Linux virtual
    // console (/dev/tty2 or similar), copy/paste operations have an effect
    // similar to TIOCSTI (CVE-2023-28100)
    {SCMP_SYS(ioctl), EPERM, &SCMP_A1(SCMP_CMP_MASKED_EQ, 0xFFFFFFFFu, (int)TIOCLINUX)},
  };

  struct f_syscall_act deny_devel[] = {
    // Profiling operations; we expect these to be done by tools from outside
    // the sandbox.  In particular perf has been the source of many CVEs.
    {SCMP_SYS(perf_event_open), EPERM},
    // Don't allow you to switch to bsd emulation or whatnot
    {SCMP_SYS(personality), EPERM, &SCMP_A0(SCMP_CMP_NE, allowed_personality)},

    {SCMP_SYS(ptrace), EPERM}
  };

  // Blocklist all but unix, inet, inet6 and netlink
  struct
  {
    int            family;
    f_syscall_opts flags_mask;
  } socket_family_allowlist[] = {
    // NOTE: Keep in numerical order
    { AF_UNSPEC, 0 },
    { AF_LOCAL, 0 },
    { AF_INET, 0 },
    { AF_INET6, 0 },
    { AF_NETLINK, 0 },
    { AF_CAN, F_CAN },
    { AF_BLUETOOTH, F_BLUETOOTH },
  };

  scmp_filter_ctx ctx = seccomp_init(SCMP_ACT_ALLOW);
  if (ctx == NULL) {
    res = 1;
    goto out;
  } else
    errno = 0;

  int ret;

  // We only really need to handle arches on multiarch systems.
  // If only one arch is supported the default is fine
  if (arch != 0) {
    // This *adds* the target arch, instead of replacing the
    // native one. This is not ideal, because we'd like to only
    // allow the target arch, but we can't really disallow the
    // native arch at this point, because then bubblewrap
    // couldn't continue running.
    ret = seccomp_arch_add(ctx, arch);
    if (ret < 0 && ret != -EEXIST) {
      res = 2;
      errno = -ret;
      goto out;
    }

    if (allow_multiarch && multiarch != 0) {
      ret = seccomp_arch_add(ctx, multiarch);
      if (ret < 0 && ret != -EEXIST) {
        res = 3;
        errno = -ret;
        goto out;
      }
    }
  }

  SECCOMP_RULESET_ADD(deny_common);
  if (opts & F_DENY_NS) SECCOMP_RULESET_ADD(deny_ns);
  if (opts & F_DENY_TTY) SECCOMP_RULESET_ADD(deny_tty);
  if (opts & F_DENY_DEVEL) SECCOMP_RULESET_ADD(deny_devel);

  if (!allow_multiarch) {
    F_println("disabling modify_ldt");

    // modify_ldt is a historic source of interesting information leaks,
    // so it's disabled as a hardening measure.
    // However, it is required to run old 16-bit applications
    // as well as some Wine patches, so it's allowed in multiarch.
    ret = seccomp_rule_add(ctx, SCMP_ACT_ERRNO(EPERM), SCMP_SYS(modify_ldt), 0);

    // See above for the meaning of EFAULT.
    if (ret == -EFAULT) {
      // call fmsg here?
      res = 4;
      goto out;
    } else if (ret < 0) {
      res = 5;
      errno = -ret;
      goto out;
    }
  }

  // Socket filtering doesn't work on e.g. i386, so ignore failures here
  // However, we need to user seccomp_rule_add_exact to avoid libseccomp doing
  // something else: https://github.com/seccomp/libseccomp/issues/8
  int last_allowed_family = -1;
  for (int i = 0; i < LEN(socket_family_allowlist); i++) {
    if (socket_family_allowlist[i].flags_mask != 0 &&
        (socket_family_allowlist[i].flags_mask & opts) != socket_family_allowlist[i].flags_mask)
      continue;

    for (int disallowed = last_allowed_family + 1; disallowed < socket_family_allowlist[i].family; disallowed++) {
      // Blocklist the in-between valid families
      seccomp_rule_add_exact(ctx, SCMP_ACT_ERRNO(EAFNOSUPPORT), SCMP_SYS(socket), 1, SCMP_A0(SCMP_CMP_EQ, disallowed));
    }
    last_allowed_family = socket_family_allowlist[i].family;
  }
  // Blocklist the rest
  seccomp_rule_add_exact(ctx, SCMP_ACT_ERRNO(EAFNOSUPPORT), SCMP_SYS(socket), 1, SCMP_A0(SCMP_CMP_GE, last_allowed_family + 1));

  ret = seccomp_export_bpf(ctx, fd);
  if (ret != 0) {
    res = 6;
    errno = -ret;
    goto out;
  }

out:
  if (ctx)
    seccomp_release(ctx);

  return res;
}
