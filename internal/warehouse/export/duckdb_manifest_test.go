package export

import (
	"strings"
	"testing"
)

func TestBuildManifestSQLUsesReadParquetList(t *testing.T) {
	sql := BuildManifestSQL("market_bars_export", []string{"/tmp/a.parquet", "/tmp/b.parquet"})
	if !strings.Contains(sql, "CREATE OR REPLACE VIEW market_bars_export AS") {
		t.Fatalf("unexpected manifest sql: %s", sql)
	}
	if !strings.Contains(sql, "read_parquet([") || !strings.Contains(sql, "/tmp/a.parquet") || !strings.Contains(sql, "/tmp/b.parquet") {
		t.Fatalf("unexpected manifest sql: %s", sql)
	}
}
