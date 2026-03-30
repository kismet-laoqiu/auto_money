package export

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/writer"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

type BarSource interface {
	LoadBars(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error)
}

type ExporterConfig struct {
	Source BarSource
	Now    func() time.Time
}

type ParquetExporter struct {
	cfg ExporterConfig
}

type ExportRequest struct {
	Provider  string   `json:"provider"`
	Symbols   []string `json:"symbols"`
	Intervals []string `json:"intervals"`
	RootDir   string   `json:"root_dir"`
}

type DatasetSummary struct {
	Provider string   `json:"provider"`
	Symbol   string   `json:"symbol"`
	Interval string   `json:"interval"`
	Rows     int      `json:"rows"`
	Files    []string `json:"files"`
	Bytes    int64    `json:"bytes"`
}

type ExportResult struct {
	GeneratedAt       time.Time        `json:"generated_at"`
	RootDir           string           `json:"root_dir"`
	LatestRootDir     string           `json:"latest_root_dir,omitempty"`
	ManifestSQLPath   string           `json:"manifest_sql_path"`
	SummaryPath       string           `json:"summary_path"`
	OverwriteStrategy string           `json:"overwrite_strategy"`
	Datasets          []DatasetSummary `json:"datasets"`
	TotalFiles        int              `json:"total_files"`
	TotalRows         int              `json:"total_rows"`
	TotalBytes        int64            `json:"total_bytes"`
}

type parquetBarRow struct {
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

func NewParquetExporter(cfg ExporterConfig) (*ParquetExporter, error) {
	if cfg.Source == nil {
		return nil, fmt.Errorf("parquet export source is nil")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &ParquetExporter{cfg: cfg}, nil
}

func (exporter *ParquetExporter) Export(ctx context.Context, request ExportRequest) (ExportResult, error) {
	if request.Provider == "" {
		request.Provider = "bitget"
	}
	if len(request.Symbols) == 0 {
		return ExportResult{}, fmt.Errorf("at least one symbol is required")
	}
	if len(request.Intervals) == 0 {
		return ExportResult{}, fmt.Errorf("at least one interval is required")
	}
	if request.RootDir == "" {
		request.RootDir = filepath.Join("artifacts", "platform", "export")
	}
	dataRoot := filepath.Join(request.RootDir, "data")
	result := ExportResult{
		GeneratedAt:       exporter.cfg.Now().UTC(),
		RootDir:           request.RootDir,
		LatestRootDir:     request.RootDir,
		ManifestSQLPath:   filepath.Join(request.RootDir, "manifest.sql"),
		SummaryPath:       filepath.Join(request.RootDir, "summary.json"),
		OverwriteStrategy: "dataset-replace",
		Datasets:          make([]DatasetSummary, 0, len(request.Symbols)*len(request.Intervals)),
	}
	allFiles := make([]string, 0)
	for _, symbol := range request.Symbols {
		for _, interval := range request.Intervals {
			bars, err := exporter.cfg.Source.LoadBars(ctx, config.DatasetConfig{Provider: request.Provider, Symbol: symbol, Interval: interval})
			if err != nil {
				return ExportResult{}, err
			}
			files, bytesWritten, err := writePartitionedParquet(ctx, bars, dataRoot, request.Provider, symbol, interval)
			if err != nil {
				return ExportResult{}, err
			}
			result.Datasets = append(result.Datasets, DatasetSummary{
				Provider: request.Provider,
				Symbol:   symbol,
				Interval: interval,
				Rows:     len(bars),
				Files:    files,
				Bytes:    bytesWritten,
			})
			result.TotalRows += len(bars)
			result.TotalFiles += len(files)
			result.TotalBytes += bytesWritten
			allFiles = append(allFiles, files...)
		}
	}
	sort.Strings(allFiles)
	if err := os.MkdirAll(request.RootDir, 0o755); err != nil {
		return ExportResult{}, err
	}
	manifest := BuildManifestSQL("market_bars_export", allFiles)
	if err := os.WriteFile(result.ManifestSQLPath, []byte(manifest), 0o644); err != nil {
		return ExportResult{}, err
	}
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return ExportResult{}, err
	}
	if err := os.WriteFile(result.SummaryPath, body, 0o644); err != nil {
		return ExportResult{}, err
	}
	return result, nil
}

func writePartitionedParquet(ctx context.Context, bars []core.Bar, root, provider, symbol, interval string) ([]string, int64, error) {
	datasetRoot := filepath.Join(root, "provider", provider, "symbol", symbol, "interval", interval)
	if err := os.RemoveAll(datasetRoot); err != nil {
		return nil, 0, err
	}
	partitions := map[string][]core.Bar{}
	dates := make([]string, 0)
	for _, bar := range bars {
		day := bar.Time.UTC().Format("2006-01-02")
		if _, ok := partitions[day]; !ok {
			dates = append(dates, day)
		}
		partitions[day] = append(partitions[day], bar)
	}
	sort.Strings(dates)
	files := make([]string, 0, len(dates))
	var totalBytes int64
	for _, day := range dates {
		filePath := filepath.Join(datasetRoot, "date", day, "bars.parquet")
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return nil, 0, err
		}
		if err := writeParquetFile(ctx, filePath, provider, symbol, interval, partitions[day]); err != nil {
			return nil, 0, err
		}
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, 0, err
		}
		files = append(files, filePath)
		totalBytes += info.Size()
	}
	return files, totalBytes, nil
}

func writeParquetFile(ctx context.Context, filePath, provider, symbol, interval string, bars []core.Bar) error {
	step, ok := intervalStep(interval)
	if !ok {
		return fmt.Errorf("unsupported export interval %s", interval)
	}
	fw, err := local.NewLocalFileWriter(filePath)
	if err != nil {
		return err
	}
	defer fw.Close()
	pw, err := writer.NewParquetWriter(fw, new(parquetBarRow), 1)
	if err != nil {
		return err
	}
	pw.CompressionType = parquet.CompressionCodec_SNAPPY
	for _, bar := range bars {
		if err := ctx.Err(); err != nil {
			return err
		}
		row := parquetBarRow{
			Provider:  provider,
			Symbol:    symbol,
			Interval:  interval,
			OpenTime:  bar.Time.UTC().UnixMilli(),
			CloseTime: bar.Time.UTC().Add(step).UnixMilli(),
			Open:      bar.Open,
			High:      bar.High,
			Low:       bar.Low,
			Close:     bar.Close,
			Volume:    bar.Volume,
		}
		if err := pw.Write(row); err != nil {
			return err
		}
	}
	return pw.WriteStop()
}

func intervalStep(interval string) (time.Duration, bool) {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "1m":
		return time.Minute, true
	case "5m":
		return 5 * time.Minute, true
	case "15m":
		return 15 * time.Minute, true
	case "1h":
		return time.Hour, true
	case "4h":
		return 4 * time.Hour, true
	case "1d":
		return 24 * time.Hour, true
	default:
		return 0, false
	}
}
