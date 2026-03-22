package models

import "testing"

func TestIsValidChatID(t *testing.T) {
	tests := []struct {
		name     string
		chatID   int64
		expected bool
	}{
		{"valid_personal", 123456789, true},
		{"valid_group", -987654321, true},
		{"valid_channel", -1001234567890, true},
		{"zero", 0, false},
		{"too_small_positive", 100, false},
		{"too_small_negative", -100, false},
		{"valid_large", 9223372036854775807, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidChatID(tt.chatID)
			if result != tt.expected {
				t.Errorf("IsValidChatID(%d) = %v, want %v", tt.chatID, result, tt.expected)
			}
		})
	}
}
