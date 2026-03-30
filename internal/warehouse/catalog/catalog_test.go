package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMigrationsOrdersFiles(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{
		"0003_init_backtest.sql": "select 3;",
		"0001_init_market.sql":   "select 1;",
		"0002_init_strategy.sql": "select 2;",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write migration %s: %v", name, err)
		}
	}
	migrations, err := LoadMigrations(root)
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	if len(migrations) != 3 {
		t.Fatalf("unexpected migration count: %+v", migrations)
	}
	if migrations[0].Name != "0001_init_market.sql" || migrations[2].Name != "0003_init_backtest.sql" {
		t.Fatalf("unexpected migration order: %+v", migrations)
	}
}
