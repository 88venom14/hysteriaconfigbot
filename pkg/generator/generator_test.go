package generator

import (
	"hysconfigbot/pkg/consts"
	"strings"
	"testing"
)

func TestGeneratePassword(t *testing.T) {
	// Проверка длины
	password, err := GeneratePassword()
	if err != nil {
		t.Fatalf("GeneratePassword() error: %v", err)
	}

	expectedLen := consts.PasswordByteLength * 2 // hex encoding
	if len(password) != expectedLen {
		t.Errorf("Password length = %d, want %d", len(password), expectedLen)
	}

	// Проверка уникальности
	passwords := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		p, err := GeneratePassword()
		if err != nil {
			t.Fatalf("GeneratePassword() iteration %d error: %v", i, err)
		}
		if passwords[p] {
			t.Error("Duplicate password generated")
		}
		passwords[p] = true
	}
}

func TestGeneratePassword_IsHex(t *testing.T) {
	password, err := GeneratePassword()
	if err != nil {
		t.Fatalf("GeneratePassword() error: %v", err)
	}

	// Проверка что пароль содержит только hex символы
	hexChars := "0123456789abcdef"
	for _, c := range password {
		if !strings.ContainsRune(hexChars, c) {
			t.Errorf("Password contains non-hex character: %c", c)
		}
	}
}

func TestGenerateConfig(t *testing.T) {
	config, err := GenerateConfig("testuser", "testpass", "example.com", 300, 300)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	// Проверка наличия ключевых элементов в конфиге
	requiredStrings := []string{
		"mixed-port: 7890",
		"server: example.com",
		"testuser:testpass",
		"up: 300",
		"down: 300",
		"⚡️ Hysteria2",
	}

	for _, req := range requiredStrings {
		if !strings.Contains(config, req) {
			t.Errorf("Config missing required string: %s", req)
		}
	}
}

func TestGenerateConfig_AutoSpeed(t *testing.T) {
	config, err := GenerateConfig("autouser", "autopass", "auto.example.com", 0, 0)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	if !strings.Contains(config, "up: 0") || !strings.Contains(config, "down: 0") {
		t.Error("Config should contain up: 0 and down: 0 for auto speed")
	}
}

func TestIsValidLatinName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid_simple", "alice", true},
		{"valid_mixed", "alice123", true},
		{"valid_uppercase", "ALICE", true},
		{"valid_mixed_case", "AliceTest123", true},
		{"empty", "", false},
		{"too_long", "thisnameiswaytoolongandshouldfail", false},
		{"cyrillic", "пользователь", false},
		{"with_underscore", "alice_bob", false},
		{"with_dash", "alice-bob", false},
		{"with_space", "alice bob", false},
		{"with_at", "alice@example", false},
		{"with_dot", "alice.bob", false},
		{"single_char", "a", true},
		{"single_digit", "7", true},
		{"max_length", "abcdefghijklmnopqrstuvwxyz123456", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidLatinName(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidLatinName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidServerAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid_domain", "example.com", true},
		{"valid_subdomain", "sub.example.com", true},
		{"valid_ip", "192.168.1.1", true},
		{"valid_with_dash", "my-server.com", true},
		{"valid_with_underscore", "my_server.com", true},
		{"empty", "", false},
		{"too_long", strings.Repeat("a", 254), false},
		{"starts_with_dash", "-example.com", false},
		{"ends_with_dash", "example.com-", false},
		{"starts_with_underscore", "_example.com", false},
		{"ends_with_underscore", "example.com_", false},
		{"double_dot", "example..com", false},
		{"template_injection", "example${cmd}com", false},
		{"with_braces", "exam{ple.com", false},
		{"with_curly", "exam}ple.com", false},
		{"with_space", "example .com", false},
		{"valid_all_chars", "a1b2-c3_d4.e5f6", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidServerAddress(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidServerAddress(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
