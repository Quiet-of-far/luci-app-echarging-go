package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsMaxRecordsPerRoomWhenMissing(t *testing.T) {
	cfg := loadConfigJSON(t, `{"rooms":[]}`)

	if cfg.MaxRecordsPerRoom != 500 {
		t.Fatalf("MaxRecordsPerRoom = %d, want 500", cfg.MaxRecordsPerRoom)
	}
}

func TestLoadPreservesZeroMaxRecordsPerRoom(t *testing.T) {
	cfg := loadConfigJSON(t, `{"rooms":[],"max_records_per_room":0}`)

	if cfg.MaxRecordsPerRoom != 0 {
		t.Fatalf("MaxRecordsPerRoom = %d, want 0", cfg.MaxRecordsPerRoom)
	}
}

func loadConfigJSON(t *testing.T, body string) *Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}
