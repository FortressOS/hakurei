package dbus

import (
	"testing"
)

func TestUnescapeValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		value   string
		want    string
		wantErr ParseError
	}{
		// upstream test cases
		{value: "abcde", want: "abcde"},
		{value: "", want: ""},
		{value: "%20%20", want: "  "},
		{value: "%24", want: "$"},
		{value: "%25", want: "%"},
		{value: "abc%24", want: "abc$"},
		{value: "%24abc", want: "$abc"},
		{value: "abc%24abc", want: "abc$abc"},
		{value: "/", want: "/"},
		{value: "-", want: "-"},
		{value: "_", want: "_"},
		{value: "A", want: "A"},
		{value: "I", want: "I"},
		{value: "Z", want: "Z"},
		{value: "a", want: "a"},
		{value: "i", want: "i"},
		{value: "z", want: "z"},
		/* Bug: https://bugs.freedesktop.org/show_bug.cgi?id=53499 */
		{value: "%c3%b6", want: "\xc3\xb6"},

		{value: "%a", wantErr: ErrBadValHexLength},
		{value: "%q", wantErr: ErrBadValHexLength},
		{value: "%az", wantErr: ErrBadValHexByte},
		{value: "%%", wantErr: ErrBadValLength},
		{value: "%$$", wantErr: ErrBadValHexByte},
		{value: "abc%a", wantErr: ErrBadValHexLength},
		{value: "%axyz", wantErr: ErrBadValHexByte},
		{value: "%", wantErr: ErrBadValLength},
		{value: "$", wantErr: ErrBadValByte},
		{value: " ", wantErr: ErrBadValByte},
	}

	for _, tc := range testCases {
		t.Run("unescape "+tc.value, func(t *testing.T) {
			t.Parallel()

			if got, errno := unescapeValue([]byte(tc.value)); errno != tc.wantErr {
				t.Errorf("unescapeValue() errno = %v, wantErr %v", errno, tc.wantErr)
			} else if tc.wantErr == errSuccess && string(got) != tc.want {
				t.Errorf("unescapeValue() = %q, want %q", got, tc.want)
			}
		})
	}
}
