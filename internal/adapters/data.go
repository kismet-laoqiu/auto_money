package adapters

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

func (client *Client) EnsureDataset(ctx context.Context, cacheDir string, spec config.DatasetConfig, refresh bool) (core.Dataset, error) {
	cachePath := datasetCachePath(cacheDir, spec)
	if !refresh {
		if bars, err := readBarsCSV(cachePath); err == nil && len(bars) > 0 {
			bars = trimBarsByRange(bars, spec.Range)
			return core.Dataset{Name: spec.Name, Provider: spec.Provider, Symbol: spec.Symbol, Interval: spec.Interval, Bars: bars}, nil
		}
	}
	bars, err := client.FetchBars(ctx, spec)
	if err != nil {
		return core.Dataset{}, err
	}
	if err := writeBarsCSV(cachePath, bars); err != nil {
		return core.Dataset{}, err
	}
	bars = trimBarsByRange(bars, spec.Range)
	return core.Dataset{Name: spec.Name, Provider: spec.Provider, Symbol: spec.Symbol, Interval: spec.Interval, Bars: bars}, nil
}

func trimBarsByRange(bars []core.Bar, rangeValue string) []core.Bar {
	if len(bars) == 0 || rangeValue == "" {
		return bars
	}
	latest := bars[len(bars)-1].Time
	cutoff := latest
	if strings.HasSuffix(rangeValue, "y") {
		years, err := strconv.Atoi(strings.TrimSuffix(rangeValue, "y"))
		if err != nil || years <= 0 {
			return bars
		}
		cutoff = latest.AddDate(-years, 0, 0)
	} else if strings.HasSuffix(rangeValue, "mo") {
		months, err := strconv.Atoi(strings.TrimSuffix(rangeValue, "mo"))
		if err != nil || months <= 0 {
			return bars
		}
		cutoff = latest.AddDate(0, -months, 0)
	} else {
		return bars
	}
	start := 0
	for start < len(bars) && bars[start].Time.Before(cutoff) {
		start++
	}
	if start >= len(bars) {
		return bars
	}
	return bars[start:]
}

func WriteReportArtifacts(root string, report core.Report) error {
	equityDir := filepath.Join(root, "equity")
	tradesDir := filepath.Join(root, "trades")
	if err := os.MkdirAll(equityDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(tradesDir, 0o755); err != nil {
		return err
	}
	prefix := sanitize(report.Name)
	if prefix == "" {
		prefix = sanitize(report.Symbol)
	}
	if err := writeEquityCSV(filepath.Join(equityDir, prefix+".csv"), report.EquityCurve); err != nil {
		return err
	}
	if err := writeTradesCSV(filepath.Join(tradesDir, prefix+".csv"), report.Trades); err != nil {
		return err
	}
	return nil
}

func datasetCachePath(root string, spec config.DatasetConfig) string {
	filename := fmt.Sprintf("%s_%s_%s.csv", sanitize(spec.Provider), sanitize(spec.Symbol), sanitize(spec.Interval))
	return filepath.Join(root, sanitize(spec.Provider), filename)
}

func sanitize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("/", "_", "=", "_", ":", "_", " ", "_", "-", "_")
	return replacer.Replace(value)
}

func writeBarsCSV(path string, bars []core.Bar) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"time", "open", "high", "low", "close", "volume"}); err != nil {
		return err
	}
	for _, bar := range bars {
		record := []string{bar.Time.UTC().Format(time.RFC3339), strconv.FormatFloat(bar.Open, 'f', -1, 64), strconv.FormatFloat(bar.High, 'f', -1, 64), strconv.FormatFloat(bar.Low, 'f', -1, 64), strconv.FormatFloat(bar.Close, 'f', -1, 64), strconv.FormatFloat(bar.Volume, 'f', -1, 64)}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return writer.Error()
}

func readBarsCSV(path string) ([]core.Bar, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	bars := make([]core.Bar, 0, len(records)-1)
	for index, record := range records {
		if index == 0 || len(record) < 6 {
			continue
		}
		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			return nil, err
		}
		openValue, _ := strconv.ParseFloat(record[1], 64)
		highValue, _ := strconv.ParseFloat(record[2], 64)
		lowValue, _ := strconv.ParseFloat(record[3], 64)
		closeValue, _ := strconv.ParseFloat(record[4], 64)
		volumeValue, _ := strconv.ParseFloat(record[5], 64)
		bars = append(bars, core.Bar{Time: timestamp, Open: openValue, High: highValue, Low: lowValue, Close: closeValue, Volume: volumeValue})
	}
	return bars, nil
}

func writeEquityCSV(path string, curve []core.EquityPoint) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"time", "equity"}); err != nil {
		return err
	}
	for _, point := range curve {
		if err := writer.Write([]string{point.Time.UTC().Format(time.RFC3339), strconv.FormatFloat(point.Equity, 'f', -1, 64)}); err != nil {
			return err
		}
	}
	return writer.Error()
}

func writeTradesCSV(path string, trades []core.Trade) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"side", "entry_time", "exit_time", "entry_price", "exit_price", "return", "bars_held", "reason", "score"}); err != nil {
		return err
	}
	for _, trade := range trades {
		record := []string{string(trade.Side), trade.EntryTime.UTC().Format(time.RFC3339), trade.ExitTime.UTC().Format(time.RFC3339), strconv.FormatFloat(trade.EntryPrice, 'f', -1, 64), strconv.FormatFloat(trade.ExitPrice, 'f', -1, 64), strconv.FormatFloat(trade.Return, 'f', -1, 64), strconv.Itoa(trade.BarsHeld), trade.Reason, strconv.FormatFloat(trade.Score, 'f', -1, 64)}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return writer.Error()
}
