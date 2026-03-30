package watchlist

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"quantlab/internal/config"
)

var DefaultHistoricalIntervals = []string{"15m", "1h", "4h", "1d", "1w"}

type File struct {
	Provider            string                    `yaml:"provider"`
	ProductType         string                    `yaml:"product_type"`
	StreamInterval      string                    `yaml:"stream_interval"`
	HistoricalIntervals []string                  `yaml:"historical_intervals"`
	HorizonDays         int                       `yaml:"horizon_days"`
	Symbols             []config.LiveSymbolConfig `yaml:"symbols"`
}

func Load(path string) (File, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read watchlist %s: %w", path, err)
	}
	var file File
	if err := yaml.Unmarshal(body, &file); err != nil {
		return File{}, fmt.Errorf("decode watchlist %s: %w", path, err)
	}
	file.applyDefaults()
	if len(file.Symbols) == 0 {
		return File{}, fmt.Errorf("watchlist %s symbols is empty", path)
	}
	return file, nil
}

func (file *File) applyDefaults() {
	if file.Provider == "" {
		file.Provider = "bitget"
	}
	if file.ProductType == "" {
		file.ProductType = "USDT-FUTURES"
	}
	if file.StreamInterval == "" {
		file.StreamInterval = "1m"
	}
	if file.HorizonDays <= 0 {
		file.HorizonDays = 365 * 3
	}
	if len(file.HistoricalIntervals) == 0 {
		file.HistoricalIntervals = append([]string(nil), DefaultHistoricalIntervals...)
		return
	}
	normalized := make([]string, 0, len(file.HistoricalIntervals))
	seen := map[string]bool{}
	for _, interval := range file.HistoricalIntervals {
		value := strings.ToLower(strings.TrimSpace(interval))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	file.HistoricalIntervals = normalized
}

func (file File) SymbolNames() []string {
	out := make([]string, 0, len(file.Symbols))
	for _, item := range file.Symbols {
		symbol := strings.ToUpper(strings.TrimSpace(item.Symbol))
		if symbol == "" {
			continue
		}
		out = append(out, symbol)
	}
	return out
}
