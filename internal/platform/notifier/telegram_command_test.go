package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCommandRuntimeRepliesToStatusAndPersistsOffset(t *testing.T) {
	telegram, platform, replies, cleanup := newCommandRuntimeTestServers(t, []telegramUpdate{{
		UpdateID: 101,
		ChatID:   "6959476905",
		UserID:   "6959476905",
		Text:     "/status",
	}}, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/status" {
			t.Fatalf("unexpected platform path: %s", request.URL.Path)
		}
		writeCommandJSON(t, writer, map[string]any{
			"last_seq": 88,
			"trader": map[string]any{
				"arming_state": "armed",
			},
		})
	})
	defer cleanup()

	store := &commandCursorStoreStub{}
	runtime := NewCommandRuntime(CommandRuntimeConfig{
		Service:         New(Config{BaseURL: telegram.URL, TelegramBotToken: "demo-token", HTTPClient: telegram.Client()}),
		StateStore:      store,
		PlatformBaseURL: platform.URL,
		AllowedChatID:   "6959476905",
		AllowedUserIDs:  []string{"6959476905"},
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if store.savedCursor != 101 {
		t.Fatalf("unexpected saved cursor: %d", store.savedCursor)
	}
	if len(*replies) != 1 {
		t.Fatalf("unexpected replies: %+v", *replies)
	}
	if (*replies)[0].ChatID != "6959476905" {
		t.Fatalf("unexpected chat id: %+v", (*replies)[0])
	}
	if !strings.Contains((*replies)[0].Text, "last_seq") || !strings.Contains((*replies)[0].Text, "armed") {
		t.Fatalf("unexpected reply text: %q", (*replies)[0].Text)
	}
}

func TestCommandRuntimeRepliesToPositions(t *testing.T) {
	telegram, platform, replies, cleanup := newCommandRuntimeTestServers(t, []telegramUpdate{{
		UpdateID: 102,
		ChatID:   "6959476905",
		UserID:   "6959476905",
		Text:     "/positions",
	}}, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/positions" {
			t.Fatalf("unexpected platform path: %s", request.URL.Path)
		}
		writeCommandJSON(t, writer, []map[string]any{{
			"symbol": "MSTRUSDT",
			"qty":    0.04,
			"status": "open",
		}})
	})
	defer cleanup()

	runtime := NewCommandRuntime(CommandRuntimeConfig{
		Service:         New(Config{BaseURL: telegram.URL, TelegramBotToken: "demo-token", HTTPClient: telegram.Client()}),
		StateStore:      &commandCursorStoreStub{},
		PlatformBaseURL: platform.URL,
		AllowedChatID:   "6959476905",
		AllowedUserIDs:  []string{"6959476905"},
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if len(*replies) != 1 || !strings.Contains((*replies)[0].Text, "MSTRUSDT") || !strings.Contains((*replies)[0].Text, "0.04") {
		t.Fatalf("unexpected replies: %+v", *replies)
	}
}

func TestCommandRuntimeRepliesToBacktestUsingResolvedConfig(t *testing.T) {
	telegram, platform, replies, cleanup := newCommandRuntimeTestServers(t, []telegramUpdate{{
		UpdateID: 103,
		ChatID:   "6959476905",
		UserID:   "6959476905",
		Text:     "/backtest mstr-wave-fib v0.1.2",
	}}, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/backtests/run" {
			t.Fatalf("unexpected platform path: %s", request.URL.Path)
		}
		if request.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", request.Method)
		}
		var body struct {
			ConfigPath string `json:"config_path"`
			Refresh    bool   `json:"refresh"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode backtest request: %v", err)
		}
		if body.ConfigPath != "configs/demo-mstr-bundle-v0.1.2.yaml" || body.Refresh {
			t.Fatalf("unexpected backtest request: %+v", body)
		}
		writeCommandJSON(t, writer, map[string]any{
			"final_score":     0.91,
			"objective_score": 0.88,
			"reports":         []map[string]any{{"dataset": "mstrusdt-1h"}},
		})
	})
	defer cleanup()

	runtime := NewCommandRuntime(CommandRuntimeConfig{
		Service:         New(Config{BaseURL: telegram.URL, TelegramBotToken: "demo-token", HTTPClient: telegram.Client()}),
		StateStore:      &commandCursorStoreStub{},
		PlatformBaseURL: platform.URL,
		AllowedChatID:   "6959476905",
		AllowedUserIDs:  []string{"6959476905"},
		ResolveConfigPath: func(strategyID, version string) (string, error) {
			if strategyID != "mstr-wave-fib" || version != "v0.1.2" {
				return "", errors.New("unexpected lookup")
			}
			return "configs/demo-mstr-bundle-v0.1.2.yaml", nil
		},
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if len(*replies) != 1 || !strings.Contains((*replies)[0].Text, "final_score") || !strings.Contains((*replies)[0].Text, "0.91") {
		t.Fatalf("unexpected replies: %+v", *replies)
	}
}

func TestCommandRuntimeRepliesToBacktestWithTelegramSafeSummary(t *testing.T) {
	const maxTelegramTextLen = 4096
	longTrades := make([]map[string]any, 0, 800)
	for i := 0; i < 800; i++ {
		longTrades = append(longTrades, map[string]any{
			"symbol":     "MSTRUSDT",
			"side":       "short",
			"entry_time": "2026-03-29T01:49:00Z",
			"exit_time":  "2026-03-29T01:50:00Z",
			"reason":     "stop_loss | SMA trend down, resistance reaction, Fib pullback zone",
		})
	}
	telegram := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/botdemo-token/getUpdates":
			writeCommandJSON(t, writer, map[string]any{
				"ok": true,
				"result": []map[string]any{{
					"update_id": 104,
					"message": map[string]any{
						"chat": map[string]any{"id": "6959476905"},
						"from": map[string]any{"id": "6959476905"},
						"text": "/backtest mstr-wave-fib v0.1.2",
					},
				}},
			})
		case "/botdemo-token/sendMessage":
			defer request.Body.Close()
			var payload struct {
				ChatID string `json:"chat_id"`
				Text   string `json:"text"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("decode telegram send payload: %v", err)
			}
			if len(payload.Text) > maxTelegramTextLen {
				writeCommandJSON(t, writer, map[string]any{"ok": false, "description": "Bad Request: message is too long"})
				return
			}
			if !strings.Contains(payload.Text, "final_score") || !strings.Contains(payload.Text, "objective_score") {
				t.Fatalf("unexpected summary text: %q", payload.Text)
			}
			writeCommandJSON(t, writer, map[string]any{"ok": true})
		default:
			t.Fatalf("unexpected telegram path: %s", request.URL.Path)
		}
	}))
	defer telegram.Close()

	platform := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/backtests/run" {
			t.Fatalf("unexpected platform path: %s", request.URL.Path)
		}
		writeCommandJSON(t, writer, map[string]any{
			"generated_at":    "2026-03-30T02:43:12.164120912Z",
			"metric_name":     "final_score",
			"final_score":     -100.09999673190664,
			"objective_score": -125.1249959148833,
			"aggregate":       map[string]any{"avg_oos_return": -0.015439715898301287, "avg_trade_count": 16, "dataset_count": 1},
			"reports":         []map[string]any{{"name": "mstrusdt_demo_replay", "symbol": "MSTRUSDT", "interval": "1m", "trades": longTrades}},
		})
	}))
	defer platform.Close()

	store := &commandCursorStoreStub{}
	runtime := NewCommandRuntime(CommandRuntimeConfig{
		Service:         New(Config{BaseURL: telegram.URL, TelegramBotToken: "demo-token", HTTPClient: telegram.Client()}),
		StateStore:      store,
		PlatformBaseURL: platform.URL,
		AllowedChatID:   "6959476905",
		AllowedUserIDs:  []string{"6959476905"},
		ResolveConfigPath: func(strategyID, version string) (string, error) {
			return "configs/demo-mstr-bundle-v0.1.2.yaml", nil
		},
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if store.savedCursor != 104 {
		t.Fatalf("unexpected saved cursor: %d", store.savedCursor)
	}
}

func TestCommandRuntimeRunRetriesTransientPollErrors(t *testing.T) {
	var polls int32
	telegram := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/botdemo-token/getUpdates" {
			t.Fatalf("unexpected telegram path: %s", request.URL.Path)
		}
		if atomic.AddInt32(&polls, 1) == 1 {
			writer.WriteHeader(http.StatusConflict)
			return
		}
		writeCommandJSON(t, writer, map[string]any{
			"ok":     true,
			"result": []map[string]any{},
		})
	}))
	defer telegram.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	runtime := NewCommandRuntime(CommandRuntimeConfig{
		Service:        New(Config{BaseURL: telegram.URL, TelegramBotToken: "demo-token", HTTPClient: telegram.Client()}),
		StateStore:     &commandCursorStoreStub{},
		AllowedChatID:  "6959476905",
		AllowedUserIDs: []string{"6959476905"},
	})

	err := runtime.Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded after retry loop, got %v", err)
	}
	if got := atomic.LoadInt32(&polls); got < 2 {
		t.Fatalf("expected runtime to retry after transient poll error, got polls=%d", got)
	}
}

type telegramUpdate struct {
	UpdateID int64
	ChatID   string
	UserID   string
	Text     string
}

type telegramReply struct {
	ChatID string
	Text   string
}

type commandCursorStoreStub struct {
	loadedCursor int64
	savedCursor  int64
}

func (store *commandCursorStoreStub) SaveConsumerCursor(_ context.Context, _ string, seq int64) error {
	store.savedCursor = seq
	return nil
}

func (store *commandCursorStoreStub) LoadConsumerCursor(_ context.Context, _ string) (int64, error) {
	return store.loadedCursor, nil
}

func newCommandRuntimeTestServers(t *testing.T, updates []telegramUpdate, platformHandler func(http.ResponseWriter, *http.Request)) (*httptest.Server, *httptest.Server, *[]telegramReply, func()) {
	t.Helper()
	replies := &[]telegramReply{}
	telegram := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/botdemo-token/getUpdates":
			writeCommandJSON(t, writer, map[string]any{
				"ok":     true,
				"result": marshalUpdates(updates),
			})
		case "/botdemo-token/sendMessage":
			defer request.Body.Close()
			var payload struct {
				ChatID string `json:"chat_id"`
				Text   string `json:"text"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("decode telegram send payload: %v", err)
			}
			*replies = append(*replies, telegramReply{ChatID: payload.ChatID, Text: payload.Text})
			writeCommandJSON(t, writer, map[string]any{"ok": true})
		default:
			t.Fatalf("unexpected telegram path: %s", request.URL.Path)
		}
	}))
	platform := httptest.NewServer(http.HandlerFunc(platformHandler))
	cleanup := func() {
		telegram.Close()
		platform.Close()
	}
	return telegram, platform, replies, cleanup
}

func marshalUpdates(updates []telegramUpdate) []map[string]any {
	result := make([]map[string]any, 0, len(updates))
	for _, update := range updates {
		result = append(result, map[string]any{
			"update_id": update.UpdateID,
			"message": map[string]any{
				"chat": map[string]any{"id": update.ChatID},
				"from": map[string]any{"id": update.UserID},
				"text": update.Text,
			},
		})
	}
	return result
}

func writeCommandJSON(t *testing.T, writer http.ResponseWriter, payload any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		t.Fatalf("encode payload: %v", err)
	}
}
