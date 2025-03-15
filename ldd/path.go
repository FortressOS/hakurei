package ldd

import (
	"path"
	"slices"
)

// Path returns a deterministic, deduplicated slice of absolute directory paths in entries.
func Path(entries []*Entry) []string {
	p := make([]string, 0, len(entries)*2)
	for _, entry := range entries {
		if path.IsAbs(entry.Path) {
			p = append(p, path.Dir(entry.Path))
		}
		if path.IsAbs(entry.Name) {
			p = append(p, path.Dir(entry.Name))
		}
	}
	slices.Sort(p)
	return slices.Compact(p)
}
