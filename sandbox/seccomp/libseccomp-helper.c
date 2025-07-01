#ifndef _GNU_SOURCE
#define _GNU_SOURCE /* CLONE_NEWUSER */
#endif

#include "libseccomp-helper.h"
#include <assert.h>
#include <errno.h>
#include <sys/socket.h>

#define LEN(arr) (sizeof(arr) / sizeof((arr)[0]))

int32_t hakurei_prepare_filter(int *ret_p, int fd, uint32_t arch,
                               uint32_t multiarch,
                               struct hakurei_syscall_rule *rules,
                               size_t rules_sz, hakurei_prepare_flag flags) {
  int i;
  int last_allowed_family;
  int disallowed;
  struct hakurei_syscall_rule *rule;

  int32_t res = 0; /* refer to resPrefix for message */

  /* Blocklist all but unix, inet, inet6 and netlink */
  struct {
    int family;
    hakurei_prepare_flag flags_mask;
  } socket_family_allowlist[] = {
      /* NOTE: Keep in numerical order */
      {AF_UNSPEC, 0},
      {AF_LOCAL, 0},
      {AF_INET, 0},
      {AF_INET6, 0},
      {AF_NETLINK, 0},
      {AF_CAN, HAKUREI_PREPARE_CAN},
      {AF_BLUETOOTH, HAKUREI_PREPARE_BLUETOOTH},
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

    if (flags & HAKUREI_PREPARE_MULTIARCH && multiarch != 0) {
      *ret_p = seccomp_arch_add(ctx, multiarch);
      if (*ret_p < 0 && *ret_p != -EEXIST) {
        res = 3;
        goto out;
      }
    }
  }

  for (i = 0; i < rules_sz; i++) {
    rule = &rules[i];
    assert(rule->m_errno == EPERM || rule->m_errno == ENOSYS);

    if (rule->arg)
      *ret_p = seccomp_rule_add(ctx, SCMP_ACT_ERRNO(rule->m_errno),
                                rule->syscall, 1, *rule->arg);
    else
      *ret_p = seccomp_rule_add(ctx, SCMP_ACT_ERRNO(rule->m_errno),
                                rule->syscall, 0);

    if (*ret_p == -EFAULT) {
      res = 4;
      goto out;
    } else if (*ret_p < 0) {
      res = 5;
      goto out;
    }
  }

  /* Socket filtering doesn't work on e.g. i386, so ignore failures here
   * However, we need to user seccomp_rule_add_exact to avoid libseccomp doing
   * something else: https://github.com/seccomp/libseccomp/issues/8 */
  last_allowed_family = -1;
  for (i = 0; i < LEN(socket_family_allowlist); i++) {
    if (socket_family_allowlist[i].flags_mask != 0 &&
        (socket_family_allowlist[i].flags_mask & flags) !=
            socket_family_allowlist[i].flags_mask)
      continue;

    for (disallowed = last_allowed_family + 1;
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
