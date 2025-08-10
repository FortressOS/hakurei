package ldd

import (
	"hakurei.app/container"
)

// Path returns a deterministic, deduplicated slice of absolute directory paths in entries.
func Path(entries []*Entry) []*container.Absolute {
	p := make([]*container.Absolute, 0, len(entries)*2)
	for _, entry := range entries {
		if a, err := container.NewAbs(entry.Path); err == nil {
			p = append(p, a.Dir())
		}
		if a, err := container.NewAbs(entry.Name); err == nil {
			p = append(p, a.Dir())
		}
	}
	container.SortAbs(p)
	return container.CompactAbs(p)
}
