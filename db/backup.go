package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupValue backs up a single value to a file.
// Used before modification if auto-backup is enabled.
func (c *DBClient) BackupValue(key string, backupDir string) (string, error) {
	val, err := c.GetValue(key)
	if err != nil {
		// If key doesn't exist, nothing to backup (e.g. new insert)
		return "", nil
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup dir: %w", err)
	}

	// Filename: key_timestamp.bak
	// Sanitize key for filename
	safeKey := sanitizeFilename(key)
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s_%s.bak", safeKey, timestamp)
	path := filepath.Join(backupDir, filename)

	if err := os.WriteFile(path, val, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return path, nil
}

func sanitizeFilename(s string) string {
	// Replace invalid chars with underscore
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalid {
		s = strings.ReplaceAll(s, char, "_")
	}
	return s
}
