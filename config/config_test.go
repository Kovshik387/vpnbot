package config

import "testing"

func TestLoadConfigRejectsInvalidAdminIDWithoutPanic(t *testing.T) {
	for _, adminID := range []string{"not-a-number", "0", "-123"} {
		t.Run(adminID, func(t *testing.T) {
			t.Setenv("BOT_TOKEN", "test-token")
			t.Setenv("ADMIN_ID", adminID)
			t.Setenv("USERNAME_API", "test-user")
			t.Setenv("PASSWORD_API", "test-password")
			t.Setenv("MARZBAN_URL", "https://example.test")
			t.Setenv("RUSSIAN_URL", "https://example.test")
			if _, err := LoadConfig(); err == nil {
				t.Fatal("expected invalid admin ID error")
			}
		})
	}
}
