package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"quantlab/internal/backtest"
	"quantlab/internal/platform/promotion"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/trader"
)

type stubStore struct {
	lastSeq    int64
	checkpoint trader.EngineState
}

func (store stubStore) LastSeq(context.Context) (int64, error) { return store.lastSeq, nil }
func (store stubStore) ListConsumerCursors(context.Context) ([]sqlitepkg.ConsumerCursorRow, error) {
	return nil, nil
}
func (store stubStore) ListLatestPositions(context.Context) ([]sqlitepkg.PositionRow, error) { return nil, nil }
func (store stubStore) ListLatestOrders(context.Context, int) ([]sqlitepkg.OrderRow, error) { return nil, nil }
func (store stubStore) ListRecentEvents(context.Context, int) ([]sqlitepkg.EventEnvelope, error) {
	return nil, nil
}
func (store stubStore) LoadCheckpoint(context.Context, string) (trader.EngineState, error) {
	return store.checkpoint, nil
}

type stubBacktests struct {
	called     bool
	configPath string
	refresh    bool
	result     backtest.Result
}

func (runner *stubBacktests) RunConfig(_ context.Context, configPath string, refresh bool) (backtest.Result, error) {
	runner.called = true
	runner.configPath = configPath
	runner.refresh = refresh
	return runner.result, nil
}

type stubPromotions struct {
	called bool
	input  promotion.CreateInput
}

func (manager *stubPromotions) CreateRequest(_ context.Context, input promotion.CreateInput) (promotion.Request, error) {
	manager.called = true
	manager.input = input
	return promotion.Request{ID: "promo-1", StrategyID: input.StrategyID, Version: input.Version, ConfigPath: input.ConfigPath, State: promotion.StateBacktestPassed}, nil
}

type stubLive struct {
	called bool
	symbol string
}

func (service *stubLive) FlattenSymbol(_ context.Context, symbol string) (map[string]any, error) {
	service.called = true
	service.symbol = symbol
	return map[string]any{"symbol": symbol, "status": "flat", "qty": 0.0}, nil
}

func TestServerInitializeListToolsAndReadStatus(t *testing.T) {
	server := NewServer(Config{
		Store:          stubStore{lastSeq: 7, checkpoint: trader.EngineState{ArmingState: trader.ArmingSafe}},
		WriteAuthToken: "secret",
	})
	responses := runServerScript(t, server,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"tester","version":"1.0.0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"quant_read_status","arguments":{}}}`,
	)
	if len(responses) != 3 {
		t.Fatalf("unexpected responses: %d", len(responses))
	}
	var initResp struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
			Capabilities struct {
				Tools struct {
					ListChanged bool `json:"listChanged"`
				} `json:"tools"`
			} `json:"capabilities"`
		} `json:"result"`
	}
	decodeLine(t, responses[0], &initResp)
	if initResp.Result.ProtocolVersion != ProtocolVersionLatest || initResp.Result.ServerInfo.Name != "quantlab-mcpd" || initResp.Result.Capabilities.Tools.ListChanged {
		t.Fatalf("unexpected initialize response: %+v", initResp)
	}
	var listResp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	decodeLine(t, responses[1], &listResp)
	if !containsTool(listResp.Result.Tools, "quant_read_status") || !containsTool(listResp.Result.Tools, "quant_write_backtest_run") {
		t.Fatalf("unexpected tools list: %+v", listResp.Result.Tools)
	}
	var callResp struct {
		Result struct {
			IsError           bool                   `json:"isError"`
			StructuredContent map[string]any         `json:"structuredContent"`
			Content           []map[string]any       `json:"content"`
		} `json:"result"`
	}
	decodeLine(t, responses[2], &callResp)
	if callResp.Result.IsError {
		t.Fatalf("unexpected tool error: %+v", callResp)
	}
	if got := int(callResp.Result.StructuredContent["last_seq"].(float64)); got != 7 {
		t.Fatalf("unexpected status payload: %+v", callResp.Result.StructuredContent)
	}
}

func TestServerWriteToolRequiresAuthToken(t *testing.T) {
	runner := &stubBacktests{result: backtest.Result{GeneratedAt: time.Unix(1710000000, 0).UTC(), FinalScore: 0.91}}
	server := NewServer(Config{Backtests: runner, WriteAuthToken: "secret"})
	responses := runServerScript(t, server,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"tester","version":"1.0.0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"quant_write_backtest_run","arguments":{"config_path":"configs/demo-mstr-bundle.yaml"}}}`,
	)
	if len(responses) != 2 {
		t.Fatalf("unexpected responses: %d", len(responses))
	}
	var callResp struct {
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	decodeLine(t, responses[1], &callResp)
	if !callResp.Result.IsError || runner.called {
		t.Fatalf("expected auth failure without backtest invocation: response=%+v called=%v", callResp, runner.called)
	}
}

func TestServerWriteToolBacktestRunAuthorized(t *testing.T) {
	runner := &stubBacktests{result: backtest.Result{GeneratedAt: time.Unix(1710000000, 0).UTC(), FinalScore: 0.91, ObjectiveScore: 0.86}}
	server := NewServer(Config{Backtests: runner, WriteAuthToken: "secret"})
	responses := runServerScript(t, server,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"tester","version":"1.0.0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"quant_write_backtest_run","arguments":{"config_path":"configs/demo-mstr-bundle.yaml","refresh":true,"auth_token":"secret"}}}`,
	)
	if len(responses) != 2 {
		t.Fatalf("unexpected responses: %d", len(responses))
	}
	var callResp struct {
		Result struct {
			IsError           bool           `json:"isError"`
			StructuredContent map[string]any `json:"structuredContent"`
		} `json:"result"`
	}
	decodeLine(t, responses[1], &callResp)
	if callResp.Result.IsError || !runner.called || runner.configPath != "configs/demo-mstr-bundle.yaml" || !runner.refresh {
		t.Fatalf("unexpected write tool result: response=%+v runner=%+v", callResp, runner)
	}
	if got := callResp.Result.StructuredContent["final_score"].(float64); got != 0.91 {
		t.Fatalf("unexpected backtest payload: %+v", callResp.Result.StructuredContent)
	}
}

func runServerScript(t *testing.T, server *Server, lines ...string) []string {
	t.Helper()
	input := strings.Join(lines, "\n") + "\n"
	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatalf("serve script: %v", err)
	}
	trimmed := strings.TrimSpace(output.String())
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func decodeLine(t *testing.T, line string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(line), target); err != nil {
		t.Fatalf("decode line %s: %v", line, err)
	}
}

func containsTool(tools []struct{ Name string "json:\"name\"" }, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}
