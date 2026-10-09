package resilience

import "testing"

func TestAssertPublicURL(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"public ipv4", "https://8.8.8.8/path", false},
		{"public ipv6", "http://[2606:4700:4700::1111]/", false},
		{"loopback v4", "http://127.0.0.1/", true},
		{"loopback v6", "http://[::1]/", true},
		{"private 10", "http://10.0.0.5/", true},
		{"private 192.168", "http://192.168.1.1/", true},
		{"private 172.16", "http://172.16.0.1/", true},
		{"link-local", "http://169.254.169.254/latest/meta-data", true}, // cloud metadata endpoint
		{"unspecified", "http://0.0.0.0/", true},
		{"multicast", "http://224.0.0.1/", true},
		{"bad scheme ftp", "ftp://8.8.8.8/", true},
		{"bad scheme file", "file:///etc/passwd", true},
		{"empty host", "http:///path", true},
		{"garbage", "://not a url", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := AssertPublicURL(c.url)
			if (err != nil) != c.wantErr {
				t.Fatalf("AssertPublicURL(%q) err=%v, wantErr=%v", c.url, err, c.wantErr)
			}
		})
	}
}
