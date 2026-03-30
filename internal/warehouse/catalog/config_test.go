package catalog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "warehouse.yaml")
	if err := os.WriteFile(path, []byte("dsn: postgres://warehouse:warehouse@127.0.0.1:55432/warehouse?sslmode=disable\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MaxOpenConns != 8 || cfg.MaxIdleConns != 4 || cfg.RetentionDays != 365 || cfg.CompressionAfterHours != 24 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.HealthTimeout != 5*time.Second {
		t.Fatalf("unexpected health timeout: %+v", cfg)
	}
}
