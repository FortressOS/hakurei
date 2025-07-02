package vfs

import "strings"

func Unmangle(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}

	v := make([]byte, len(s))
	var (
		j int
		c byte
	)
	for i := 0; i < len(s); i++ {
		c = s[i]
		if c == '\\' && len(s) > i+3 &&
			(s[i+1] == '0' || s[i+1] == '1') &&
			(s[i+2] >= '0' && s[i+2] <= '7') &&
			(s[i+3] >= '0' && s[i+3] <= '7') {
			c = ((s[i+1] - '0') << 6) |
				((s[i+2] - '0') << 3) |
				(s[i+3] - '0')
			i += 3
		}
		v[j] = c
		j++
	}
	return string(v[:j])
}
