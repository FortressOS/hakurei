package vfs_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"

	"git.gensokyo.uk/security/hakurei/sandbox/vfs"
)

func TestUnfold(t *testing.T) {
	testCases := []struct {
		name    string
		sample  string
		target  string
		wantErr error

		want         *vfs.MountInfoNode
		wantCollectF func(n *vfs.MountInfoNode) []*vfs.MountInfoNode
		wantCollectN []string
	}{
		{
			"no match",
			sampleMountinfoBase,
			"/mnt",
			syscall.ESTALE, nil, nil, nil,
		},
		{
			"cover",
			`33 1 0:33 / / rw,relatime shared:1 - tmpfs impure rw,size=16777216k,mode=755
37 33 0:32 / /proc rw,nosuid,nodev,noexec,relatime shared:41 - proc proc rw
551 33 0:121 / /mnt rw,relatime shared:666 - tmpfs tmpfs rw
595 551 0:123 / /mnt rw,relatime shared:990 - tmpfs tmpfs rw
611 595 0:142 / /mnt/etc rw,relatime shared:1112 - tmpfs tmpfs rw
625 644 0:142 /passwd /mnt/etc/passwd rw,relatime shared:1112 - tmpfs tmpfs rw
641 625 0:33 /etc/passwd /mnt/etc/passwd rw,relatime shared:1 - tmpfs impure rw,size=16777216k,mode=755
644 611 0:33 /etc/passwd /mnt/etc/passwd rw,relatime shared:1 - tmpfs impure rw,size=16777216k,mode=755
`, "/mnt", nil,
			mn(595, 551, 0, 123, "/", "/mnt", "rw,relatime", o("shared:990"), "tmpfs", "tmpfs", "rw", false,
				mn(611, 595, 0, 142, "/", "/mnt/etc", "rw,relatime", o("shared:1112"), "tmpfs", "tmpfs", "rw", false,
					mn(644, 611, 0, 33, "/etc/passwd", "/mnt/etc/passwd", "rw,relatime", o("shared:1"), "tmpfs", "impure", "rw,size=16777216k,mode=755", true,
						mn(625, 644, 0, 142, "/passwd", "/mnt/etc/passwd", "rw,relatime", o("shared:1112"), "tmpfs", "tmpfs", "rw", true,
							mn(641, 625, 0, 33, "/etc/passwd", "/mnt/etc/passwd", "rw,relatime", o("shared:1"), "tmpfs", "impure", "rw,size=16777216k,mode=755", false,
								nil, nil), nil), nil), nil), nil), func(n *vfs.MountInfoNode) []*vfs.MountInfoNode {
				return []*vfs.MountInfoNode{n, n.FirstChild, n.FirstChild.FirstChild.FirstChild.FirstChild}
			}, []string{"/mnt", "/mnt/etc", "/mnt/etc/passwd"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := vfs.NewMountInfoDecoder(strings.NewReader(tc.sample))
			got, err := d.Unfold(tc.target)

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Unfold: error = %v, wantErr %v",
					err, tc.wantErr)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Unfold:\ngot  %s\nwant %s",
					mustMarshal(got), mustMarshal(tc.want))
			}

			if err == nil && tc.wantCollectF != nil {
				t.Run("collective", func(t *testing.T) {
					wantCollect := tc.wantCollectF(got)
					gotCollect := slices.Collect(got.Collective())
					if !reflect.DeepEqual(gotCollect, wantCollect) {
						t.Errorf("Collective: \ngot  %#v\nwant %#v",
							gotCollect, wantCollect)
					}
					t.Run("target", func(t *testing.T) {
						gotCollectN := slices.Collect[string](func(yield func(v string) bool) {
							for _, cur := range gotCollect {
								if !yield(cur.Clean) {
									return
								}
							}
						})
						if !reflect.DeepEqual(gotCollectN, tc.wantCollectN) {
							t.Errorf("Collective: got %q, want %q",
								gotCollectN, tc.wantCollectN)
						}
					})
				})
			}
		})
	}
}
