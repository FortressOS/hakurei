#include <seccomp.h>
#include <stdint.h>

#if (SCMP_VER_MAJOR < 2) || (SCMP_VER_MAJOR == 2 && SCMP_VER_MINOR < 5) || \
    (SCMP_VER_MAJOR == 2 && SCMP_VER_MINOR == 5 && SCMP_VER_MICRO < 1)
#error This package requires libseccomp >= v2.5.1
#endif

typedef enum {
  HAKUREI_EXPORT_MULTIARCH = 1 << 0,
  HAKUREI_EXPORT_CAN = 1 << 1,
  HAKUREI_EXPORT_BLUETOOTH = 1 << 2,
} hakurei_export_flag;

struct hakurei_syscall_rule {
  int syscall;
  int m_errno;
  struct scmp_arg_cmp *arg;
};

extern void *hakurei_scmp_allocate(uintptr_t f, size_t len);
int32_t hakurei_scmp_make_filter(
    int *ret_p, uintptr_t allocate_p,
    uint32_t arch, uint32_t multiarch,
    struct hakurei_syscall_rule *rules,
    size_t rules_sz, hakurei_export_flag flags);
