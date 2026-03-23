package client

import (
	"strings"
	"testing"
)

func TestIsValidProxyAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Разрешённые localhost
		{"localhost_with_port", "127.0.0.1:2080", true},
		{"localhost_name", "localhost:7890", true},
		{"localhost_only", "127.0.0.1", true},
		{"localhost_name_only", "localhost", true},

		// Запрещённые private IP
		{"private_10", "10.0.0.1:9050", false},
		{"private_192", "192.168.1.1:9050", false},
		{"private_172_16", "172.16.0.1:9050", false},
		{"private_172_31", "172.31.255.255:9050", false},
		{"link_local_ipv4", "169.254.1.1:9050", false},
		{"ipv6_localhost", "::1:9050", false},
		{"ipv6_link_local", "fe80::1:9050", false},
		{"all_zeros", "0.0.0.0:9050", false},

		// Публичные адреса (должны быть разрешены)
		{"public_ip", "203.0.113.1:9050", true},
		{"public_domain", "proxy.example.com:9050", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidProxyAddress(tt.input)
			if result != tt.expected {
				t.Errorf("isValidProxyAddress(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeProxyURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with_password", "http://user:password@proxy.com:8080", "http://user:***@proxy.com:8080"},
		{"without_password", "http://user@proxy.com:8080", "http://user@proxy.com:8080"},
		{"no_auth", "http://proxy.com:8080", "http://proxy.com:8080"},
		{"invalid_url", "not a valid url", "not%20a%20valid%20url"},
		{"empty", "", ""},
		{"socks5_with_auth", "socks5://admin:secret123@tor-proxy:9050", "socks5://admin:***@tor-proxy:9050"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeProxyURL(tt.input)
			// Проверяем, что пароль скрыт (звёздочки могут быть URL-encoded как %2A%2A%2A)
			if tt.input == "http://user:password@proxy.com:8080" {
				if !strings.Contains(result, "***") && !strings.Contains(result, "%2A%2A%2A") {
					t.Errorf("sanitizeProxyURL(%q) should hide password, got %q", tt.input, result)
				}
				return
			}
			if tt.input == "socks5://admin:secret123@tor-proxy:9050" {
				if !strings.Contains(result, "***") && !strings.Contains(result, "%2A%2A%2A") {
					t.Errorf("sanitizeProxyURL(%q) should hide password, got %q", tt.input, result)
				}
				return
			}
			if result != tt.expected {
				t.Errorf("sanitizeProxyURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
