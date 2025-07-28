#include "shim-signal.h"
#include <errno.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

static pid_t hakurei_shim_param_ppid = -1;

// this cannot unblock hlog since Go code is not async-signal-safe
static void hakurei_shim_sigaction(int sig, siginfo_t *si, void *ucontext) {
  if (sig != SIGCONT || si == NULL) {
    // unreachable
    fprintf(stderr, "sigaction: sa_sigaction got invalid siginfo\n");
    return;
  }

  // monitor requests shim exit
  if (si->si_pid == hakurei_shim_param_ppid)
    exit(254);

  fprintf(stderr, "sigaction: got SIGCONT from process %d\n", si->si_pid);

  // shim orphaned before monitor delivers a signal
  if (getppid() != hakurei_shim_param_ppid)
    exit(3);
}

void hakurei_shim_setup_cont_signal(pid_t ppid) {
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
}
