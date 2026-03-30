package watchlistsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"quantlab/internal/config"
	basewatchlist "quantlab/internal/watchlist"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type Config struct {
	WatchlistPath       string
	LiveConfigPath      string
	WarehouseConfigPath string
	PlatformctlPath     string
	MarketdRestartUnit  string
	Runner              Runner
	Now                 func() time.Time
}

type Service struct {
	cfg Config
}

type ApplyResult struct {
	AppliedAt       time.Time       `json:"applied_at"`
	WatchlistPath   string          `json:"watchlist_path"`
	Symbols         []string        `json:"symbols"`
	RestartUnit     string          `json:"restart_unit"`
	PlatformctlJSON json.RawMessage `json:"platformctl_json"`
}

func NewService(cfg Config) *Service {
	if cfg.PlatformctlPath == "" {
		cfg.PlatformctlPath = "./bin/platformctl"
	}
	if cfg.MarketdRestartUnit == "" {
		cfg.MarketdRestartUnit = "quantlab-marketd.service"
	}
	if cfg.Runner == nil {
		cfg.Runner = commandRunner{}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{cfg: cfg}
}

func (service *Service) Load() (basewatchlist.File, error) {
	return basewatchlist.Load(service.cfg.WatchlistPath)
}

func (service *Service) SymbolsText() (string, error) {
	file, err := service.Load()
	if err != nil {
		return "", err
	}
	return strings.Join(file.SymbolNames(), "\n"), nil
}

func (service *Service) SaveSymbols(raw string) (basewatchlist.File, error) {
	file, err := service.Load()
	if err != nil {
		return basewatchlist.File{}, err
	}
	symbols := parseSymbols(raw)
	if len(symbols) == 0 {
		return basewatchlist.File{}, fmt.Errorf("symbols are empty")
	}
	existing := make(map[string]config.LiveSymbolConfig, len(file.Symbols))
	for _, item := range file.Symbols {
		name := strings.ToUpper(strings.TrimSpace(item.Symbol))
		if name == "" {
			continue
		}
		existing[name] = item
	}
	defaults := config.LiveSymbolConfig{MaxNotional: 500, MaxTranches: 4}
	for _, item := range file.Symbols {
		if item.MaxNotional > 0 {
			defaults.MaxNotional = item.MaxNotional
		}
		if item.MaxTranches > 0 {
			defaults.MaxTranches = item.MaxTranches
		}
		break
	}
	file.Symbols = make([]config.LiveSymbolConfig, 0, len(symbols))
	for _, symbol := range symbols {
		item, ok := existing[symbol]
		if !ok {
			item = defaults
			item.Symbol = symbol
		}
		item.Symbol = symbol
		file.Symbols = append(file.Symbols, item)
	}
	body, err := yaml.Marshal(&file)
	if err != nil {
		return basewatchlist.File{}, err
	}
	if err := os.WriteFile(service.cfg.WatchlistPath, body, 0o644); err != nil {
		return basewatchlist.File{}, err
	}
	return file, nil
}

func (service *Service) Apply(ctx context.Context) (ApplyResult, error) {
	file, err := service.Load()
	if err != nil {
		return ApplyResult{}, err
	}
	output, err := service.cfg.Runner.Run(
		ctx,
		service.cfg.PlatformctlPath,
		"watchlist", "apply",
		"-watchlist", service.cfg.WatchlistPath,
		"-live-config", service.cfg.LiveConfigPath,
		"-warehouse-config", service.cfg.WarehouseConfigPath,
	)
	if err != nil {
		return ApplyResult{}, err
	}
	if _, err := service.cfg.Runner.Run(ctx, "systemctl", "restart", service.cfg.MarketdRestartUnit); err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{
		AppliedAt:       service.cfg.Now().UTC(),
		WatchlistPath:   service.cfg.WatchlistPath,
		Symbols:         file.SymbolNames(),
		RestartUnit:     service.cfg.MarketdRestartUnit,
		PlatformctlJSON: append([]byte(nil), output...),
	}, nil
}

func parseSymbols(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, field := range fields {
		symbol := strings.ToUpper(strings.TrimSpace(field))
		if symbol == "" || seen[symbol] {
			continue
		}
		seen[symbol] = true
		out = append(out, symbol)
	}
	return out
}

type commandRunner struct{}

func (commandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("%s %v failed: %s", name, args, message)
	}
	return output, nil
}
