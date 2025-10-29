#include <signal.h>

/* see shim.go for documentation */
typedef enum {
  HAKUREI_SHIM_EXIT_REQUESTED,
  HAKUREI_SHIM_ORPHAN,
  HAKUREI_SHIM_INVALID,
  HAKUREI_SHIM_BAD_PID,
} hakurei_shim_msg;

void hakurei_shim_setup_cont_signal(pid_t ppid, int fd);
