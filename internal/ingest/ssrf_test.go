package ingest

import (
	"net"
	"testing"
)

func TestValidatePublicHTTPURL_RejectsPrivateAndBadSchemes(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "http loopback ip", url: "http://127.0.0.1/video.mp4", wantErr: true},
		{name: "https private ip", url: "https://192.168.1.10/x", wantErr: true},
		{name: "link local", url: "http://169.254.169.254/", wantErr: true},
		{name: "file scheme", url: "file:///etc/passwd", wantErr: true},
		{name: "ftp scheme", url: "ftp://example.com/a", wantErr: true},
		{name: "missing host", url: "https:///path", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePublicHTTPURL(tc.url)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %s", tc.url)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.url, err)
			}
		})
	}
}

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{"127.0.0.1", "::1", "10.0.0.1", "192.168.0.1", "169.254.1.1", "0.0.0.0"}
	for _, s := range blocked {
		ip := net.ParseIP(s)
		if ip == nil || !isBlockedIP(ip) {
			t.Fatalf("expected blocked: %s", s)
		}
	}
	if isBlockedIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("8.8.8.8 should not be blocked")
	}
}
