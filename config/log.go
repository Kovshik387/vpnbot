package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// SetupLog перенаправляет стандартный log в файл, если задан BOT_LOG_PATH в .env.
func SetupLog() error {
	path := strings.TrimSpace(os.Getenv("BOT_LOG_PATH"))
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("каталог для лога: %w", err)
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("файл лога: %w", err)
	}
	log.SetOutput(f)
	return nil
}
