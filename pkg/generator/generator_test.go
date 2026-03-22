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
