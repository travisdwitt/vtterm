package table

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func SaveDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".vtterm")
}

func Save(t *Table) error {
	t.UpdatedAt = time.Now()
	dir := SaveDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating save directory: %w", err)
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling table: %w", err)
	}

	path := filepath.Join(dir, sanitize(t.Name)+".json")
	return os.WriteFile(path, data, 0o644)
}

func Load(path string) (*Table, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Table
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func ListSaved() ([]string, error) {
	dir := SaveDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

var nonAlpha = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func sanitize(name string) string {
	s := nonAlpha.ReplaceAllString(strings.ToLower(name), "_")
	s = strings.Trim(s, "_")
	if s == "" {
		s = "untitled"
	}
	return s
}
