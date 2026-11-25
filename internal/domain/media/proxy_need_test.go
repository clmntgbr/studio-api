package media

import "testing"

func TestNeedsProxyEncode(t *testing.T) {
	cases := []struct {
		w, h int
		want bool
	}{
		{1280, 720, false},
		{720, 1280, false},
		{640, 360, false},
		{1920, 1080, true},
		{1080, 1920, true},
		{0, 720, true},
		{1280, 0, true},
	}
	for _, tc := range cases {
		if got := NeedsProxyEncode(tc.w, tc.h); got != tc.want {
			t.Fatalf("NeedsProxyEncode(%d,%d)=%v want %v", tc.w, tc.h, got, tc.want)
		}
	}
}
