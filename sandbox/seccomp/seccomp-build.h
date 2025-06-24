#include <seccomp.h>
#include <stdint.h>

#if (SCMP_VER_MAJOR < 2) || (SCMP_VER_MAJOR == 2 && SCMP_VER_MINOR < 5) ||     \
    (SCMP_VER_MAJOR == 2 && SCMP_VER_MINOR == 5 && SCMP_VER_MICRO < 1)
#error This package requires libseccomp >= v2.5.1
#endif

typedef enum {
  HAKUREI_VERBOSE = 1 << 0,
  HAKUREI_EXT = 1 << 1,
  HAKUREI_DENY_NS = 1 << 2,
  HAKUREI_DENY_TTY = 1 << 3,
  HAKUREI_DENY_DEVEL = 1 << 4,
  HAKUREI_MULTIARCH = 1 << 5,
  HAKUREI_LINUX32 = 1 << 6,
  HAKUREI_CAN = 1 << 7,
  HAKUREI_BLUETOOTH = 1 << 8,
} hakurei_filter_opts;

extern void hakurei_println(char *v);
int32_t hakurei_build_filter(int *ret_p, int fd, uint32_t arch, uint32_t multiarch,
                       hakurei_filter_opts opts);