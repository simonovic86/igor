// SPDX-License-Identifier: Apache-2.0

package hostcall

import (
	"testing"

	"github.com/simonovic86/igor/pkg/manifest"
)

func TestExtractAllowedHosts(t *testing.T) {
	tests := []struct {
		name string
		cfg  manifest.CapabilityConfig
		want []string
	}{
		{
			name: "nil options",
			cfg:  manifest.CapabilityConfig{},
			want: nil,
		},
		{
			name: "no allowed_hosts key",
			cfg:  manifest.CapabilityConfig{Options: map[string]any{"timeout_ms": 5000}},
			want: nil,
		},
		{
			name: "valid hosts",
			cfg: manifest.CapabilityConfig{
				Options: map[string]any{
					"allowed_hosts": []any{"api.example.com", "httpbin.org"},
				},
			},
			want: []string{"api.example.com", "httpbin.org"},
		},
		{
			name: "mixed types filtered",
			cfg: manifest.CapabilityConfig{
				Options: map[string]any{
					"allowed_hosts": []any{"valid.com", 123, "also-valid.com"},
				},
			},
			want: []string{"valid.com", "also-valid.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAllowedHosts(tt.cfg)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("host[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCheckAllowedHost(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		hosts   []string
		wantErr bool
	}{
		{
			name:    "empty allowlist permits all",
			url:     "https://anything.com/path",
			hosts:   nil,
			wantErr: false,
		},
		{
			name:    "allowed host",
			url:     "https://api.example.com/v1/data",
			hosts:   []string{"api.example.com"},
			wantErr: false,
		},
		{
			name:    "blocked host",
			url:     "https://evil.com/steal",
			hosts:   []string{"api.example.com"},
			wantErr: true,
		},
		{
			name:    "case insensitive",
			url:     "https://API.Example.COM/v1",
			hosts:   []string{"api.example.com"},
			wantErr: false,
		},
		{
			name:    "port stripped from hostname",
			url:     "https://api.example.com:8443/v1",
			hosts:   []string{"api.example.com"},
			wantErr: false,
		},
		{
			name:    "invalid URL",
			url:     "://not-a-url",
			hosts:   []string{"anything.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAllowedHost(tt.url, tt.hosts)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkAllowedHost(%q, %v) error = %v, wantErr %v", tt.url, tt.hosts, err, tt.wantErr)
			}
		})
	}
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want map[string]string
	}{
		{
			name: "single header",
			raw:  "Content-Type: application/json\n",
			want: map[string]string{"Content-Type": "application/json"},
		},
		{
			name: "multiple headers",
			raw:  "Authorization: Bearer token\nAccept: text/plain\n",
			want: map[string]string{"Authorization": "Bearer token", "Accept": "text/plain"},
		},
		{
			name: "empty string",
			raw:  "",
			want: map[string]string{},
		},
		{
			name: "trailing newlines",
			raw:  "X-Custom: value\n\n\n",
			want: map[string]string{"X-Custom": "value"},
		},
		{
			name: "no colon skipped",
			raw:  "garbage\nX-Valid: yes\n",
			want: map[string]string{"X-Valid": "yes"},
		},
		{
			name: "value with colon",
			raw:  "Authorization: Basic dXNlcjpwYXNz\n",
			want: map[string]string{"Authorization": "Basic dXNlcjpwYXNz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHeaders(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for k, wantV := range tt.want {
				if gotV, ok := got[k]; !ok || gotV != wantV {
					t.Errorf("header[%q] = %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}

func TestExtractIntOption(t *testing.T) {
	tests := []struct {
		name       string
		opts       map[string]any
		key        string
		defaultVal int
		want       int
	}{
		{"nil opts", nil, "timeout_ms", 10000, 10000},
		{"missing key", map[string]any{"other": 5}, "timeout_ms", 10000, 10000},
		{"float64 value", map[string]any{"timeout_ms": float64(5000)}, "timeout_ms", 10000, 5000},
		{"int value", map[string]any{"timeout_ms": 3000}, "timeout_ms", 10000, 3000},
		{"wrong type", map[string]any{"timeout_ms": "fast"}, "timeout_ms", 10000, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIntOption(tt.opts, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
