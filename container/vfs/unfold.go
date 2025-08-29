package vfs

import (
	"iter"
	"path"
	"strings"
)

type UnfoldTargetError string

func (e UnfoldTargetError) Error() string {
	return "mount point " + string(e) + " never appeared in mountinfo"
}

// MountInfoNode positions a [MountInfoEntry] in its mount hierarchy.
type MountInfoNode struct {
	*MountInfoEntry
	FirstChild  *MountInfoNode `json:"first_child"`
	NextSibling *MountInfoNode `json:"next_sibling"`

	Clean   string `json:"clean"`
	Covered bool   `json:"covered"`
}

// Collective returns an iterator over visible mountinfo nodes.
func (n *MountInfoNode) Collective() iter.Seq[*MountInfoNode] {
	return func(yield func(*MountInfoNode) bool) { n.visit(yield) }
}

func (n *MountInfoNode) visit(yield func(*MountInfoNode) bool) bool {
	if !n.Covered && !yield(n) {
		return false
	}
	for cur := n.FirstChild; cur != nil; cur = cur.NextSibling {
		if !cur.visit(yield) {
			return false
		}
	}
	return true
}

// Unfold unfolds the mount hierarchy and resolves covered paths.
func (d *MountInfoDecoder) Unfold(target string) (*MountInfoNode, error) {
	targetClean := path.Clean(target)

	var mountinfoSize int
	for range d.Entries() {
		mountinfoSize++
	}
	if err := d.Err(); err != nil {
		return nil, err
	}

	mountinfo := make([]*MountInfoNode, mountinfoSize)
	// mount ID to index lookup
	idIndex := make(map[int]int, mountinfoSize)
	// final entry to match target
	targetIndex := -1
	{
		i := 0
		for ent := range d.Entries() {
			mountinfo[i] = &MountInfoNode{Clean: path.Clean(ent.Target), MountInfoEntry: ent}
			idIndex[ent.ID] = i
			if mountinfo[i].Clean == targetClean {
				targetIndex = i
			}

			i++
		}
	}

	if targetIndex == -1 {
		// target does not exist in parsed mountinfo
		return nil, &DecoderError{Op: "unfold", Line: -1, Err: UnfoldTargetError(targetClean)}
	}

	for _, cur := range mountinfo {
		var parent *MountInfoNode
		if p, ok := idIndex[cur.Parent]; !ok {
			continue
		} else {
			parent = mountinfo[p]
		}

		if !strings.HasPrefix(cur.Clean, targetClean) {
			continue
		}
		if parent.Clean == cur.Clean {
			parent.Covered = true
		}

		covered := false
		nsp := &parent.FirstChild
		for s := parent.FirstChild; s != nil; s = s.NextSibling {
			if strings.HasPrefix(cur.Clean, s.Clean) {
				covered = true
				break
			}

			if strings.HasPrefix(s.Clean, cur.Clean) {
				*nsp = s.NextSibling
			} else {
				nsp = &s.NextSibling
			}
		}
		if covered {
			continue
		}
		*nsp = cur
	}

	return mountinfo[targetIndex], nil
}
