package export

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/reader"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

type parquetTestRow struct {
	Provider  string  `parquet:"name=provider, type=BYTE_ARRAY, convertedtype=UTF8"`
	Symbol    string  `parquet:"name=symbol, type=BYTE_ARRAY, convertedtype=UTF8"`
	Interval  string  `parquet:"name=interval, type=BYTE_ARRAY, convertedtype=UTF8"`
	OpenTime  int64   `parquet:"name=open_time, type=INT64, convertedtype=TIMESTAMP_MILLIS"`
	CloseTime int64   `parquet:"name=close_time, type=INT64, convertedtype=TIMESTAMP_MILLIS"`
	Open      float64 `parquet:"name=open, type=DOUBLE"`
	High      float64 `parquet:"name=high, type=DOUBLE"`
	Low       float64 `parquet:"name=low, type=DOUBLE"`
	Close     float64 `parquet:"name=close, type=DOUBLE"`
	Volume    float64 `parquet:"name=volume, type=DOUBLE"`
}

func TestParquetExporterWritesPartitionedFilesAndManifest(t *testing.T) {
	base := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	source := &sourceStub{bars: map[string][]core.Bar{
		"bitget:BTCUSDT:1h": {
			{Time: base, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10},
			{Time: base.Add(1 * time.Hour), Open: 1.5, High: 3, Low: 1.2, Close: 2.5, Volume: 11},
		},
		"bitget:ETHUSDT:1h": {
			{Time: base, Open: 10, High: 12, Low: 9, Close: 11, Volume: 20},
			{Time: base.Add(1 * time.Hour), Open: 11, High: 13, Low: 10, Close: 12, Volume: 21},
		},
	}}
	root := t.TempDir()
	exporter, err := NewParquetExporter(ExporterConfig{Source: source})
	if err != nil {
		t.Fatalf("new exporter: %v", err)
	}
	result, err := exporter.Export(context.Background(), ExportRequest{
		Provider:  "bitget",
		Symbols:   []string{"BTCUSDT", "ETHUSDT"},
		Intervals: []string{"1h"},
		RootDir:   root,
	})
	if err != nil {
		t.Fatalf("export parquet: %v", err)
	}
	if len(result.Datasets) != 2 {
		t.Fatalf("unexpected datasets: %+v", result.Datasets)
	}
	wantPartition := filepath.Join("provider", "bitget", "symbol", "BTCUSDT", "interval", "1h", "date", "2026-03-29")
	if !strings.Contains(result.Datasets[0].Files[0], wantPartition) {
		t.Fatalf("unexpected partition path: %s", result.Datasets[0].Files[0])
	}
	if _, err := os.Stat(result.ManifestSQLPath); err != nil {
		t.Fatalf("stat manifest sql: %v", err)
	}
	if _, err := os.Stat(result.SummaryPath); err != nil {
		t.Fatalf("stat summary json: %v", err)
	}
	manifestBody, err := os.ReadFile(result.ManifestSQLPath)
	if err != nil {
		t.Fatalf("read manifest sql: %v", err)
	}
	if !strings.Contains(string(manifestBody), "CREATE OR REPLACE VIEW market_bars_export AS") || !strings.Contains(string(manifestBody), "bars.parquet") {
		t.Fatalf("unexpected manifest sql: %s", string(manifestBody))
	}
	firstDatasetRows := readParquetRows(t, result.Datasets[0].Files[0])
	if len(firstDatasetRows) != 2 {
		t.Fatalf("unexpected parquet row count: %d", len(firstDatasetRows))
	}
	if firstDatasetRows[0].Provider != "bitget" || firstDatasetRows[0].Symbol != "BTCUSDT" || firstDatasetRows[0].Interval != "1h" {
		t.Fatalf("unexpected first parquet row: %+v", firstDatasetRows[0])
	}
	if firstDatasetRows[0].OpenTime != base.UnixMilli() || firstDatasetRows[0].CloseTime != base.Add(time.Hour).UnixMilli() {
		t.Fatalf("unexpected parquet timestamps: %+v", firstDatasetRows[0])
	}
	totalRows := 0
	for _, dataset := range result.Datasets {
		for _, file := range dataset.Files {
			totalRows += len(readParquetRows(t, file))
		}
	}
	if totalRows != 4 {
		t.Fatalf("unexpected total parquet rows: %d", totalRows)
	}
	body, err := os.ReadFile(result.SummaryPath)
	if err != nil {
		t.Fatalf("read summary json: %v", err)
	}
	var written ExportResult
	if err := json.Unmarshal(body, &written); err != nil {
		t.Fatalf("unmarshal summary json: %v", err)
	}
	if written.TotalRows != 4 || written.TotalFiles != 2 {
		t.Fatalf("unexpected summary body: %+v", written)
	}
}

func readParquetRows(t *testing.T, file string) []parquetTestRow {
	t.Helper()
	fr, err := local.NewLocalFileReader(file)
	if err != nil {
		t.Fatalf("open parquet file: %v", err)
	}
	defer fr.Close()
	pr, err := reader.NewParquetReader(fr, new(parquetTestRow), 1)
	if err != nil {
		t.Fatalf("new parquet reader: %v", err)
	}
	defer pr.ReadStop()
	rows := make([]parquetTestRow, int(pr.GetNumRows()))
	if err := pr.Read(&rows); err != nil {
		t.Fatalf("read parquet rows: %v", err)
	}
	return rows
}

type sourceStub struct {
	bars map[string][]core.Bar
}

func (stub *sourceStub) LoadBars(_ context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	key := spec.Provider + ":" + spec.Symbol + ":" + spec.Interval
	return append([]core.Bar(nil), stub.bars[key]...), nil
}
