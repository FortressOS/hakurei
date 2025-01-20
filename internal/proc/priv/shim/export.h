#include <stdint.h>
#include <seccomp.h>

#if (SCMP_VER_MAJOR < 2) || \
    (SCMP_VER_MAJOR == 2 && SCMP_VER_MINOR < 5) || \
    (SCMP_VER_MAJOR == 2 && SCMP_VER_MINOR == 5 && SCMP_VER_MICRO < 1)
#error This package requires libseccomp >= v2.5.1
#endif

typedef enum {
  F_DENY_NS    = 1 << 0,
  F_DENY_TTY   = 1 << 1,
  F_DENY_DEVEL = 1 << 2,
  F_MULTIARCH  = 1 << 3,
  F_LINUX32    = 1 << 4,
  F_CAN        = 1 << 5,
  F_BLUETOOTH  = 1 << 6,
} f_syscall_opts;

extern void F_println(char *v);
int f_tmpfile_fd();
int32_t f_export_bpf(int fd, uint32_t arch, uint32_t multiarch, f_syscall_opts opts);