package ldd

import (
	"hakurei.app/container/check"
)

// Path returns a deterministic, deduplicated slice of absolute directory paths in entries.
func Path(entries []*Entry) []*check.Absolute {
	p := make([]*check.Absolute, 0, len(entries)*2)
	for _, entry := range entries {
		if a, err := check.NewAbs(entry.Path); err == nil {
			p = append(p, a.Dir())
		}
		if a, err := check.NewAbs(entry.Name); err == nil {
			p = append(p, a.Dir())
		}
	}
	check.SortAbs(p)
	return check.CompactAbs(p)
}
