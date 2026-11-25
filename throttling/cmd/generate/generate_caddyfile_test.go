package main

import (
	"strings"
	"testing"
)

func TestParseRulesAndGenerateCaddyfile(t *testing.T) {
	rules, err := parseRules(strings.NewReader(`
# One shared limiter for both SEC hosts.
sec 8r/s sec.gov *.sec.gov
market 30r/m query.example.com
`))
	if err != nil {
		t.Fatal(err)
	}

	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rules))
	}

	generated := generateCaddyfile(rules)
	for _, expected := range []string{
		"log_skip @healthz",
		"basic_auth", "{$PROXY_USER:test-proxy-user}",
		"request_header -Authorization",
		"jitter 0.1", "zone sec", "host sec.gov *.sec.gov", "key sec", "events 8",
		"window 1s", "@allowed host sec.gov *.sec.gov query.example.com",
		"reverse_proxy @allowed {host}:443", "tls_server_name {host}",
		"header_down -Retry-After-Ms", "network_proxy none", "respond 403",
	} {
		if !strings.Contains(generated, expected) {
			t.Errorf("generated Caddyfile does not contain %q", expected)
		}
	}
}

func TestParseRulesRejectsInvalidPolicies(t *testing.T) {
	tests := []struct {
		name   string
		policy string
		want   string
	}{
		{name: "empty", policy: "# none\n", want: "at least one rule"},
		{name: "short", policy: "sec 8r/s\n", want: "expected bucket"},
		{name: "bucket", policy: "SEC 8r/s sec.gov\n", want: "invalid bucket"},
		{name: "rate", policy: "sec 0r/s sec.gov\n", want: "invalid rate"},
		{name: "host", policy: "sec 8r/s localhost\n", want: "invalid host"},
		{
			name:   "duplicate bucket",
			policy: "sec 8r/s sec.gov\nsec 5r/s data.sec.gov\n",
			want:   "duplicate bucket",
		},
		{
			name:   "duplicate host",
			policy: "one 8r/s sec.gov\ntwo 5r/s sec.gov\n",
			want:   "already belongs",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseRules(strings.NewReader(test.policy))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got error %v, want text %q", err, test.want)
			}
		})
	}
}

func TestParseRate(t *testing.T) {
	events, window, err := parseRate("100r/h")
	if err != nil {
		t.Fatal(err)
	}
	if events != 100 || window != "1h" {
		t.Fatalf("got %d events in %s, want 100 events in 1h", events,
			window)
	}
}
