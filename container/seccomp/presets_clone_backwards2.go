//go:build s390 || s390x

package seccomp

/* Architectures with CONFIG_CLONE_BACKWARDS2: the child stack
 * and flags arguments are reversed so the flags come second */
const cloneArg = 1
