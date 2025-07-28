#include "shim-signal.h"
#include <errno.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

static pid_t hakurei_shim_param_ppid = -1;
static int hakurei_shim_fd = -1;

static ssize_t hakurei_shim_write(const void *buf, size_t count) {
  int savedErrno = errno;
  ssize_t ret = write(hakurei_shim_fd, buf, count);
  if (ret == -1 && errno != EAGAIN)
    exit(EXIT_FAILURE);
  errno = savedErrno;
  return ret;
}

/* see shim_linux.go for handling of the value */
static void hakurei_shim_sigaction(int sig, siginfo_t *si, void *ucontext) {
  if (sig != SIGCONT || si == NULL) {
    /* unreachable */
    hakurei_shim_write("\2", 1);
    return;
  }

  if (si->si_pid == hakurei_shim_param_ppid) {
    /* monitor requests shim exit */
    hakurei_shim_write("\0", 1);
    return;
  }

  /* unexpected si_pid */
  hakurei_shim_write("\3", 1);

  if (getppid() != hakurei_shim_param_ppid)
    /* shim orphaned before monitor delivers a signal */
    hakurei_shim_write("\1", 1);
}

void hakurei_shim_setup_cont_signal(pid_t ppid, int fd) {
  if (hakurei_shim_param_ppid != -1 || hakurei_shim_fd != -1)
    *(int *)NULL = 0; /* unreachable */

  struct sigaction new_action = {0}, old_action = {0};
  if (sigaction(SIGCONT, NULL, &old_action) != 0)
    return;
  if (old_action.sa_handler != SIG_DFL) {
    errno = ENOTRECOVERABLE;
    return;
  }

  new_action.sa_sigaction = hakurei_shim_sigaction;
  if (sigemptyset(&new_action.sa_mask) != 0)
    return;
  new_action.sa_flags = SA_ONSTACK | SA_SIGINFO;

  if (sigaction(SIGCONT, &new_action, NULL) != 0)
    return;

  errno = 0;
  hakurei_shim_param_ppid = ppid;
  hakurei_shim_fd = fd;
}
