package std

import "iter"

// Syscalls returns an iterator over all wired syscalls.
func Syscalls() iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		for name, num := range syscallNum {
			if !yield(name, num) {
				return
			}
		}
		for name, num := range syscallNumExtra {
			if !yield(name, num) {
				return
			}
		}
	}
}

// SyscallResolveName resolves a syscall number from its string representation.
func SyscallResolveName(name string) (num int, ok bool) {
	if num, ok = syscallNum[name]; ok {
		return
	}
	num, ok = syscallNumExtra[name]
	return
}
