package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"quantlab/internal/strategybundle"
)

type CommandCursorStore interface {
	SaveConsumerCursor(ctx context.Context, consumer string, seq int64) error
	LoadConsumerCursor(ctx context.Context, consumer string) (int64, error)
}

type CommandRuntimeConfig struct {
	ConsumerKey       string
	StateStore        CommandCursorStore
	Service           *Service
	PlatformBaseURL   string
	AllowedChatID     string
	AllowedUserIDs    []string
	ConfigDir         string
	PollTimeout       time.Duration
	ResolveConfigPath func(strategyID, version string) (string, error)
}

type CommandRuntime struct {
	cfg    CommandRuntimeConfig
	client *http.Client
	offset int64
	loaded bool
}

type CommandHealth struct {
	ConsumerKey     string   `json:"consumer_key"`
	Offset          int64    `json:"offset"`
	Loaded          bool     `json:"loaded"`
	Enabled         bool     `json:"enabled"`
	AllowedChatID   string   `json:"allowed_chat_id"`
	AllowedUserIDs  []string `json:"allowed_user_ids"`
	PlatformBaseURL string   `json:"platform_base_url"`
	ConfigDir       string   `json:"config_dir"`
}

type TelegramID string

const (
	commandRuntimeErrorBackoff = time.Second
	telegramMaxMessageChars    = 4096
	telegramTruncationSuffix   = "\n...(truncated)"
)

type telegramCommandUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  struct {
		Chat struct {
			ID TelegramID `json:"id"`
		} `json:"chat"`
		From struct {
			ID TelegramID `json:"id"`
		} `json:"from"`
		Text string `json:"text"`
	} `json:"message"`
}

func NewCommandRuntime(cfg CommandRuntimeConfig) *CommandRuntime {
	if cfg.ConsumerKey == "" {
		cfg.ConsumerKey = "telegram.command"
	}
	if cfg.PlatformBaseURL == "" {
		cfg.PlatformBaseURL = "http://127.0.0.1:8080"
	}
	if cfg.ConfigDir == "" {
		cfg.ConfigDir = "configs"
	}
	if cfg.PollTimeout <= 0 {
		cfg.PollTimeout = 15 * time.Second
	}
	if cfg.ResolveConfigPath == nil {
		configDir := cfg.ConfigDir
		cfg.ResolveConfigPath = func(strategyID, version string) (string, error) {
			return ResolveBundleConfigPath(configDir, strategyID, version)
		}
	}
	client := http.DefaultClient
	if cfg.Service != nil && cfg.Service.client != nil {
		client = cfg.Service.client
	}
	return &CommandRuntime{cfg: cfg, client: client}
}

func (runtime *CommandRuntime) Enabled() bool {
	return runtime != nil && runtime.cfg.Service != nil && runtime.cfg.Service.cfg.TelegramBotToken != "" && runtime.cfg.AllowedChatID != ""
}

func (runtime *CommandRuntime) Health() CommandHealth {
	return CommandHealth{
		ConsumerKey:     runtime.cfg.ConsumerKey,
		Offset:          runtime.offset,
		Loaded:          runtime.loaded,
		Enabled:         runtime.Enabled(),
		AllowedChatID:   runtime.cfg.AllowedChatID,
		AllowedUserIDs:  append([]string(nil), runtime.cfg.AllowedUserIDs...),
		PlatformBaseURL: runtime.cfg.PlatformBaseURL,
		ConfigDir:       runtime.cfg.ConfigDir,
	}
}

func (runtime *CommandRuntime) Run(ctx context.Context) error {
	for {
		if err := runtime.ProcessAvailable(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(commandRuntimeErrorBackoff):
			}
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func (runtime *CommandRuntime) ProcessAvailable(ctx context.Context) error {
	if !runtime.Enabled() {
		return nil
	}
	if err := runtime.ensureLoaded(ctx); err != nil {
		return err
	}
	updates, err := runtime.cfg.Service.ListTelegramUpdates(ctx, runtime.offset+1, runtime.cfg.PollTimeout)
	if err != nil {
		return err
	}
	for _, update := range updates {
		if runtime.isAllowed(update) {
			reply, err := runtime.handleCommand(ctx, strings.TrimSpace(update.Message.Text))
			if err != nil {
				reply = fmt.Sprintf("命令失败\n%s", err.Error())
			}
			if strings.TrimSpace(reply) != "" {
				if err := runtime.cfg.Service.SendTelegramMessage(ctx, string(update.Message.Chat.ID), reply); err != nil {
					return err
				}
			}
		}
		runtime.offset = update.UpdateID
		if runtime.cfg.StateStore != nil {
			if err := runtime.cfg.StateStore.SaveConsumerCursor(ctx, runtime.cfg.ConsumerKey, runtime.offset); err != nil {
				return err
			}
		}
	}
	return nil
}

func (runtime *CommandRuntime) ensureLoaded(ctx context.Context) error {
	if runtime.loaded {
		return nil
	}
	if runtime.cfg.StateStore == nil {
		runtime.loaded = true
		return nil
	}
	offset, err := runtime.cfg.StateStore.LoadConsumerCursor(ctx, runtime.cfg.ConsumerKey)
	if err != nil {
		return err
	}
	runtime.offset = offset
	runtime.loaded = true
	return nil
}

func (runtime *CommandRuntime) isAllowed(update telegramCommandUpdate) bool {
	if strings.TrimSpace(update.Message.Text) == "" {
		return false
	}
	if runtime.cfg.AllowedChatID != "" && string(update.Message.Chat.ID) != runtime.cfg.AllowedChatID {
		return false
	}
	if len(runtime.cfg.AllowedUserIDs) > 0 && !slices.Contains(runtime.cfg.AllowedUserIDs, string(update.Message.From.ID)) {
		return false
	}
	return true
}

func (runtime *CommandRuntime) handleCommand(ctx context.Context, text string) (string, error) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", nil
	}
	switch fields[0] {
	case "/status":
		return runtime.fetchPlatformJSON(ctx, http.MethodGet, "/api/status", nil)
	case "/positions":
		return runtime.fetchPlatformJSON(ctx, http.MethodGet, "/api/positions", nil)
	case "/backtest":
		if len(fields) != 3 {
			return "用法\n/backtest <strategy_id> <version>", nil
		}
		configPath, err := runtime.cfg.ResolveConfigPath(fields[1], fields[2])
		if err != nil {
			return "", err
		}
		body, err := json.Marshal(map[string]any{
			"config_path": configPath,
			"refresh":     false,
		})
		if err != nil {
			return "", err
		}
		return runtime.fetchPlatformBacktestSummary(ctx, body)
	default:
		return strings.Join([]string{
			"支持命令",
			"/status",
			"/positions",
			"/backtest <strategy_id> <version>",
		}, "\n"), nil
	}
}

func (runtime *CommandRuntime) fetchPlatformJSON(ctx context.Context, method, path string, body []byte) (string, error) {
	payload, err := runtime.fetchPlatformPayload(ctx, method, path, body)
	if err != nil {
		return "", err
	}
	return formatCommandJSON(payload)
}

func (runtime *CommandRuntime) fetchPlatformBacktestSummary(ctx context.Context, body []byte) (string, error) {
	payload, err := runtime.fetchPlatformPayload(ctx, http.MethodPost, "/api/backtests/run", body)
	if err != nil {
		return "", err
	}
	return formatBacktestCommandJSON(payload)
}

func (runtime *CommandRuntime) fetchPlatformPayload(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(runtime.cfg.PlatformBaseURL, "/")+path, reader)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := runtime.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("platform api status=%d body=%s", response.StatusCode, strings.TrimSpace(string(payload)))
	}
	return payload, nil
}

func formatCommandJSON(payload []byte) (string, error) {
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return "", err
	}
	pretty, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return "", err
	}
	return limitTelegramText(string(pretty)), nil
}

func formatBacktestCommandJSON(payload []byte) (string, error) {
	var decoded struct {
		GeneratedAt    string  `json:"generated_at"`
		MetricName     string  `json:"metric_name"`
		FinalScore     float64 `json:"final_score"`
		ObjectiveScore float64 `json:"objective_score"`
		Aggregate      struct {
			AvgOOSReturn  float64 `json:"avg_oos_return"`
			AvgTradeCount float64 `json:"avg_trade_count"`
			DatasetCount  int     `json:"dataset_count"`
		} `json:"aggregate"`
		Reports []struct {
			Name     string `json:"name"`
			Symbol   string `json:"symbol"`
			Interval string `json:"interval"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return "", err
	}
	lines := []string{
		"backtest completed",
		fmt.Sprintf("generated_at: %s", decoded.GeneratedAt),
		fmt.Sprintf("metric_name: %s", decoded.MetricName),
		fmt.Sprintf("final_score: %.6f", decoded.FinalScore),
		fmt.Sprintf("objective_score: %.6f", decoded.ObjectiveScore),
		fmt.Sprintf("avg_oos_return: %.6f", decoded.Aggregate.AvgOOSReturn),
		fmt.Sprintf("avg_trade_count: %.0f", decoded.Aggregate.AvgTradeCount),
		fmt.Sprintf("dataset_count: %d", decoded.Aggregate.DatasetCount),
	}
	if len(decoded.Reports) > 0 {
		report := decoded.Reports[0]
		lines = append(lines, fmt.Sprintf("report: %s %s %s", report.Name, report.Symbol, report.Interval))
	}
	return limitTelegramText(strings.Join(lines, "\n")), nil
}

func limitTelegramText(text string) string {
	runes := []rune(text)
	if len(runes) <= telegramMaxMessageChars {
		return text
	}
	suffix := []rune(telegramTruncationSuffix)
	limit := telegramMaxMessageChars - len(suffix)
	if limit < 0 {
		limit = 0
	}
	return string(runes[:limit]) + telegramTruncationSuffix
}

func ResolveBundleConfigPath(configDir, strategyID, version string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(configDir, "*.yaml"))
	if err != nil {
		return "", err
	}
	for _, path := range matches {
		_, bundle, err := strategybundle.LoadConfig(path)
		if err != nil || bundle == nil {
			continue
		}
		if bundle.StrategyID == strategyID && bundle.Version == version {
			return path, nil
		}
	}
	return "", fmt.Errorf("bundle config not found for strategy=%s version=%s", strategyID, version)
}

func (id *TelegramID) UnmarshalJSON(data []byte) error {
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		*id = TelegramID(asString)
		return nil
	}
	var asNumber json.Number
	if err := json.Unmarshal(data, &asNumber); err != nil {
		return err
	}
	*id = TelegramID(asNumber.String())
	return nil
}
