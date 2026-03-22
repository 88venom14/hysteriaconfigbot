package generator

import "testing"

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "alice", "alice"},
		{"with_spaces", "alice bob", "alice_bob"},
		{"path_traversal", "../../../etc/passwd", "etcpasswd"},
		{"backslash", "user\\name", "username"},
		{"null_byte", "user\x00name", "username"},
		{"special_chars", "user@name#", "user_name_"},
		{"empty", "", "config"},
		{"only_special", "@#$", "___"},
		{"cyrillic", "пользователь", "____________"},
		{"mixed", "user123_test", "user123_test"},
		{"with_dots", "user.name", "user_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFileName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFileName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
