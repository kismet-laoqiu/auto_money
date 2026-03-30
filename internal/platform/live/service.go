package live

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"

	"quantlab/internal/exchange/bitget"
)

type Runner interface {
	Run(ctx context.Context, execdPath string, args ...string) error
}

type PositionReader interface {
	FetchSinglePosition(ctx context.Context, symbol, productType, marginCoin string) (bitget.SinglePositionSnapshot, error)
}

type Config struct {
	ExecdPath      string
	ConfigPath     string
	StateDBPath    string
	ProductType    string
	MarginCoin     string
	AllowedSymbols []string
	Runner         Runner
	Reader         PositionReader
	Now            func() time.Time
}

type FlattenResult struct {
	Symbol      string    `json:"symbol"`
	Status      string    `json:"status"`
	Qty         float64   `json:"qty"`
	ProductType string    `json:"product_type"`
	MarginCoin  string    `json:"margin_coin"`
	CheckedAt   time.Time `json:"checked_at"`
}

type Service struct {
	cfg     Config
	allowed map[string]struct{}
}

func NewService(cfg Config) *Service {
	if cfg.ExecdPath == "" {
		cfg.ExecdPath = "./execd"
	}
	if cfg.ProductType == "" {
		cfg.ProductType = "USDT-FUTURES"
	}
	if cfg.MarginCoin == "" {
		cfg.MarginCoin = "USDT"
	}
	if cfg.Runner == nil {
		cfg.Runner = commandRunner{}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	allowed := make(map[string]struct{}, len(cfg.AllowedSymbols))
	for _, symbol := range cfg.AllowedSymbols {
		allowed[symbol] = struct{}{}
	}
	return &Service{cfg: cfg, allowed: allowed}
}

func (service *Service) FlattenSymbol(ctx context.Context, symbol string) (FlattenResult, error) {
	if symbol == "" {
		return FlattenResult{}, fmt.Errorf("symbol is required")
	}
	if len(service.allowed) > 0 {
		if _, ok := service.allowed[symbol]; !ok {
			return FlattenResult{}, fmt.Errorf("symbol %s is not allowed", symbol)
		}
	}
	if service.cfg.ConfigPath == "" {
		return FlattenResult{}, fmt.Errorf("config path is empty")
	}
	if service.cfg.Runner == nil {
		return FlattenResult{}, fmt.Errorf("execd runner is nil")
	}
	if service.cfg.Reader == nil {
		return FlattenResult{}, fmt.Errorf("position reader is nil")
	}
	args := []string{"-config", service.cfg.ConfigPath}
	if service.cfg.StateDBPath != "" {
		args = append(args, "-state-db", service.cfg.StateDBPath)
	}
	args = append(args, "-flatten-symbol", symbol)
	if err := service.cfg.Runner.Run(ctx, service.cfg.ExecdPath, args...); err != nil {
		return FlattenResult{}, err
	}
	position, err := service.cfg.Reader.FetchSinglePosition(ctx, symbol, service.cfg.ProductType, service.cfg.MarginCoin)
	if err != nil {
		return FlattenResult{}, err
	}
	status := "open"
	if math.Abs(position.Qty) < 1e-9 {
		status = "flat"
	}
	return FlattenResult{
		Symbol:      symbol,
		Status:      status,
		Qty:         position.Qty,
		ProductType: service.cfg.ProductType,
		MarginCoin:  service.cfg.MarginCoin,
		CheckedAt:   service.cfg.Now().UTC(),
	}, nil
}

type commandRunner struct{}

func (commandRunner) Run(ctx context.Context, execdPath string, args ...string) error {
	command := exec.CommandContext(ctx, execdPath, args...)
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		return err
	}
	return fmt.Errorf("execd failed: %s", message)
}
