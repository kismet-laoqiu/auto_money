package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"quantlab/internal/backtest"
	"quantlab/internal/platform/live"
	"quantlab/internal/platform/promotion"
	"quantlab/internal/platform/query"
	"quantlab/internal/platform/truth"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/strategybundle"
	"quantlab/internal/trader"
)

const (
	ProtocolVersionLatest = "2025-06-18"
	protocolVersionCompat = "2024-11-05"
)

type Reader interface {
	LastSeq(context.Context) (int64, error)
	ListConsumerCursors(context.Context) ([]sqlitepkg.ConsumerCursorRow, error)
	ListLatestPositions(context.Context) ([]sqlitepkg.PositionRow, error)
	ListLatestOrders(context.Context, int) ([]sqlitepkg.OrderRow, error)
	ListRecentEvents(context.Context, int) ([]sqlitepkg.EventEnvelope, error)
	LoadCheckpoint(context.Context, string) (trader.EngineState, error)
}

type QueryService interface {
	BarsFromConfig(ctx context.Context, configPath, datasetName string, refresh bool) (query.BarsResult, error)
	FeaturesFromConfig(ctx context.Context, configPath, datasetName string, refresh bool, offset int) (query.FeaturesResult, error)
}

type StrategyRegistry interface {
	List(strategyID string) ([]strategybundle.VersionInfo, error)
}

type BacktestRunner interface {
	RunConfig(ctx context.Context, configPath string, refresh bool) (backtest.Result, error)
}

type PromotionManager interface {
	CreateRequest(ctx context.Context, input promotion.CreateInput) (promotion.Request, error)
}

type LiveOps interface {
	FlattenSymbol(ctx context.Context, symbol string) (live.FlattenResult, error)
}

type TruthService interface {
	SiteFacts(ctx context.Context) (truth.SiteFacts, error)
	OperatorPolicy(ctx context.Context) (truth.OperatorPolicy, error)
	Leaders(ctx context.Context) (truth.LeadersFile, error)
	LeaderScore(ctx context.Context, address string) (truth.Leader, error)
	UpsertLeaderScore(ctx context.Context, input truth.LeaderScoreInput) (truth.Leader, error)
}

type Config struct {
	Store          Reader
	Query          QueryService
	Strategies     StrategyRegistry
	Backtests      BacktestRunner
	Promotions     PromotionManager
	Live           LiveOps
	Truth          TruthService
	WriteAuthToken string
	ServerName     string
	ServerVersion  string
}

type Server struct {
	cfg        Config
	negotiated bool
	ready      bool
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDef struct {
	Name        string
	Title       string
	Description string
	InputSchema map[string]any
	ReadOnly    bool
	Handler     func(context.Context, map[string]any) (map[string]any, error)
}

func NewServer(cfg Config) *Server {
	if cfg.ServerName == "" {
		cfg.ServerName = "quantlab-mcpd"
	}
	if cfg.ServerVersion == "" {
		cfg.ServerVersion = "0.1.0"
	}
	return &Server{cfg: cfg}
}

func (server *Server) Serve(ctx context.Context, reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	encoder := json.NewEncoder(writer)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		response, ok := server.handleLine(ctx, []byte(line))
		if !ok {
			continue
		}
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (server *Server) handleLine(ctx context.Context, line []byte) (rpcResponse, bool) {
	var request rpcRequest
	if err := json.Unmarshal(line, &request); err != nil {
		return rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}}, true
	}
	if request.JSONRPC != "2.0" || request.Method == "" {
		return server.errorResponse(request.ID, -32600, "invalid request"), len(request.ID) > 0
	}
	if request.Method == "initialize" {
		return server.handleInitialize(request), true
	}
	if request.Method == "notifications/initialized" {
		server.ready = true
		return rpcResponse{}, false
	}
	if !server.negotiated || !server.ready {
		return server.errorResponse(request.ID, -32002, "server not initialized"), len(request.ID) > 0
	}
	switch request.Method {
	case "tools/list":
		return server.handleToolsList(request.ID), true
	case "tools/call":
		return server.handleToolsCall(ctx, request)
	default:
		return server.errorResponse(request.ID, -32601, "method not found"), len(request.ID) > 0
	}
}

func (server *Server) handleInitialize(request rpcRequest) rpcResponse {
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(request.Params, &params)
	server.negotiated = true
	server.ready = false
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]any{
			"protocolVersion": server.protocolVersion(params.ProtocolVersion),
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    server.cfg.ServerName,
				"title":   "QuantLab MCPD",
				"version": server.cfg.ServerVersion,
			},
			"instructions": "QuantLab control-plane MCP. Read tools expose runtime and research data. Write tools require auth_token and still execute through the platform control plane rather than direct exchange credentials.",
		},
	}
}

func (server *Server) handleToolsList(id json.RawMessage) rpcResponse {
	tools := server.toolDefinitions()
	payload := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		payload = append(payload, map[string]any{
			"name":        tool.Name,
			"title":       tool.Title,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
			"annotations": map[string]any{"readOnlyHint": tool.ReadOnly},
		})
	}
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: map[string]any{"tools": payload}}
}

func (server *Server) handleToolsCall(ctx context.Context, request rpcRequest) (rpcResponse, bool) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return server.errorResponse(request.ID, -32602, "invalid params"), true
	}
	for _, tool := range server.toolDefinitions() {
		if tool.Name != params.Name {
			continue
		}
		result, err := tool.Handler(ctx, params.Arguments)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: request.ID, Result: toolErrorResult(err)}, true
		}
		return rpcResponse{JSONRPC: "2.0", ID: request.ID, Result: toolSuccessResult(result)}, true
	}
	return server.errorResponse(request.ID, -32601, "tool not found"), true
}

func (server *Server) toolDefinitions() []toolDef {
	tools := []toolDef{
		{
			Name:        "quant_read_status",
			Title:       "Quant Read Status",
			Description: "Read the current runtime status, cursors, and trader checkpoint.",
			InputSchema: objectSchema(nil, nil),
			ReadOnly:    true,
			Handler:     server.handleReadStatus,
		},
		{
			Name:        "quant_read_positions",
			Title:       "Quant Read Positions",
			Description: "Read the latest positions snapshot from runtime state.",
			InputSchema: objectSchema(nil, nil),
			ReadOnly:    true,
			Handler:     server.handleReadPositions,
		},
		{
			Name:        "quant_read_orders",
			Title:       "Quant Read Orders",
			Description: "Read recent orders with an optional limit.",
			InputSchema: objectSchema(map[string]any{"limit": map[string]any{"type": "integer", "minimum": 1}}, nil),
			ReadOnly:    true,
			Handler:     server.handleReadOrders,
		},
		{
			Name:        "quant_read_events",
			Title:       "Quant Read Events",
			Description: "Read recent event log envelopes with an optional limit.",
			InputSchema: objectSchema(map[string]any{"limit": map[string]any{"type": "integer", "minimum": 1}}, nil),
			ReadOnly:    true,
			Handler:     server.handleReadEvents,
		},
		{
			Name:        "quant_read_bars",
			Title:       "Quant Read Bars",
			Description: "Load bars from a strategy config and dataset name.",
			InputSchema: objectSchema(map[string]any{"config_path": stringSchema(), "dataset": stringSchema(), "refresh": boolSchema()}, []string{"config_path"}),
			ReadOnly:    true,
			Handler:     server.handleReadBars,
		},
		{
			Name:        "quant_read_features",
			Title:       "Quant Read Features",
			Description: "Extract the latest or offset feature set from a strategy config and dataset name.",
			InputSchema: objectSchema(map[string]any{"config_path": stringSchema(), "dataset": stringSchema(), "refresh": boolSchema(), "offset": map[string]any{"type": "integer", "minimum": 0}}, []string{"config_path"}),
			ReadOnly:    true,
			Handler:     server.handleReadFeatures,
		},
		{
			Name:        "quant_read_strategy_versions",
			Title:       "Quant Read Strategy Versions",
			Description: "List known strategy versions from the registry.",
			InputSchema: objectSchema(map[string]any{"strategy_id": stringSchema()}, nil),
			ReadOnly:    true,
			Handler:     server.handleReadStrategyVersions,
		},
		{
			Name:        "quant_read_site_facts",
			Title:       "Quant Read Site Facts",
			Description: "Read repo truth-layer site facts for the current ECS runtime.",
			InputSchema: objectSchema(nil, nil),
			ReadOnly:    true,
			Handler:     server.handleReadSiteFacts,
		},
		{
			Name:        "quant_read_operator_policy",
			Title:       "Quant Read Operator Policy",
			Description: "Read operator write-policy and channel rules.",
			InputSchema: objectSchema(nil, nil),
			ReadOnly:    true,
			Handler:     server.handleReadOperatorPolicy,
		},
		{
			Name:        "quant_read_leaders",
			Title:       "Quant Read Leaders",
			Description: "Read configured leader entries.",
			InputSchema: objectSchema(nil, nil),
			ReadOnly:    true,
			Handler:     server.handleReadLeaders,
		},
		{
			Name:        "quant_read_leader_score",
			Title:       "Quant Read Leader Score",
			Description: "Read one leader score by address.",
			InputSchema: objectSchema(map[string]any{"address": stringSchema()}, []string{"address"}),
			ReadOnly:    true,
			Handler:     server.handleReadLeaderScore,
		},
	}
	if server.cfg.WriteAuthToken == "" {
		return tools
	}
	return append(tools,
		toolDef{
			Name:        "quant_write_backtest_run",
			Title:       "Quant Write Backtest Run",
			Description: "Run a backtest through the control plane. Requires auth_token.",
			InputSchema: objectSchema(map[string]any{"config_path": stringSchema(), "refresh": boolSchema(), "auth_token": stringSchema()}, []string{"config_path", "auth_token"}),
			ReadOnly:    false,
			Handler:     server.handleWriteBacktestRun,
		},
		toolDef{
			Name:        "quant_write_promotion_request",
			Title:       "Quant Write Promotion Request",
			Description: "Create a promotion request through the control plane. Requires auth_token.",
			InputSchema: objectSchema(map[string]any{"strategy_id": stringSchema(), "version": stringSchema(), "config_path": stringSchema(), "refresh": boolSchema(), "auth_token": stringSchema()}, []string{"strategy_id", "version", "config_path", "auth_token"}),
			ReadOnly:    false,
			Handler:     server.handleWritePromotionRequest,
		},
		toolDef{
			Name:        "quant_write_live_flatten",
			Title:       "Quant Write Live Flatten",
			Description: "Flatten an allowed live symbol through execd. Requires auth_token.",
			InputSchema: objectSchema(map[string]any{"symbol": stringSchema(), "auth_token": stringSchema()}, []string{"symbol", "auth_token"}),
			ReadOnly:    false,
			Handler:     server.handleWriteLiveFlatten,
		},
		toolDef{
			Name:        "quant_write_leader_score",
			Title:       "Quant Write Leader Score",
			Description: "Upsert one leader score into leaders.yaml. Requires auth_token.",
			InputSchema: objectSchema(map[string]any{
				"address":    stringSchema(),
				"label":      stringSchema(),
				"status":     stringSchema(),
				"total":      map[string]any{"type": "number"},
				"grade":      stringSchema(),
				"note":       stringSchema(),
				"components": map[string]any{"type": "object"},
				"auth_token": stringSchema(),
			}, []string{"address", "total", "auth_token"}),
			ReadOnly: false,
			Handler:  server.handleWriteLeaderScore,
		},
	)
}

func (server *Server) handleReadStatus(ctx context.Context, _ map[string]any) (map[string]any, error) {
	if server.cfg.Store == nil {
		return nil, fmt.Errorf("status store is nil")
	}
	lastSeq, err := server.cfg.Store.LastSeq(ctx)
	if err != nil {
		return nil, err
	}
	cursors, err := server.cfg.Store.ListConsumerCursors(ctx)
	if err != nil {
		return nil, err
	}
	checkpoint, err := server.cfg.Store.LoadCheckpoint(ctx, "trader.runtime")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"last_seq": lastSeq,
		"cursors":  cursors,
		"trader": map[string]any{
			"arming_state": checkpoint.ArmingState,
			"symbols":      checkpoint.Symbols,
		},
	}, nil
}

func (server *Server) handleReadPositions(ctx context.Context, _ map[string]any) (map[string]any, error) {
	if server.cfg.Store == nil {
		return nil, fmt.Errorf("status store is nil")
	}
	positions, err := server.cfg.Store.ListLatestPositions(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"positions": positions}, nil
}

func (server *Server) handleReadOrders(ctx context.Context, args map[string]any) (map[string]any, error) {
	if server.cfg.Store == nil {
		return nil, fmt.Errorf("status store is nil")
	}
	limit, err := intArg(args, "limit", 20)
	if err != nil {
		return nil, err
	}
	orders, err := server.cfg.Store.ListLatestOrders(ctx, limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"orders": orders}, nil
}

func (server *Server) handleReadEvents(ctx context.Context, args map[string]any) (map[string]any, error) {
	if server.cfg.Store == nil {
		return nil, fmt.Errorf("status store is nil")
	}
	limit, err := intArg(args, "limit", 50)
	if err != nil {
		return nil, err
	}
	events, err := server.cfg.Store.ListRecentEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"events": events}, nil
}

func (server *Server) handleReadBars(ctx context.Context, args map[string]any) (map[string]any, error) {
	if server.cfg.Query == nil {
		return nil, fmt.Errorf("query service is nil")
	}
	configPath, err := requiredStringArg(args, "config_path")
	if err != nil {
		return nil, err
	}
	dataset, _ := optionalStringArg(args, "dataset")
	refresh, err := boolArg(args, "refresh", false)
	if err != nil {
		return nil, err
	}
	result, err := server.cfg.Query.BarsFromConfig(ctx, configPath, dataset, refresh)
	if err != nil {
		return nil, err
	}
	return map[string]any{"bars": result}, nil
}

func (server *Server) handleReadFeatures(ctx context.Context, args map[string]any) (map[string]any, error) {
	if server.cfg.Query == nil {
		return nil, fmt.Errorf("query service is nil")
	}
	configPath, err := requiredStringArg(args, "config_path")
	if err != nil {
		return nil, err
	}
	dataset, _ := optionalStringArg(args, "dataset")
	refresh, err := boolArg(args, "refresh", false)
	if err != nil {
		return nil, err
	}
	offset, err := intArg(args, "offset", 0)
	if err != nil {
		return nil, err
	}
	result, err := server.cfg.Query.FeaturesFromConfig(ctx, configPath, dataset, refresh, offset)
	if err != nil {
		return nil, err
	}
	return map[string]any{"features": result}, nil
}

func (server *Server) handleReadStrategyVersions(_ context.Context, args map[string]any) (map[string]any, error) {
	if server.cfg.Strategies == nil {
		return nil, fmt.Errorf("strategy registry is nil")
	}
	strategyID, _ := optionalStringArg(args, "strategy_id")
	versions, err := server.cfg.Strategies.List(strategyID)
	if err != nil {
		return nil, err
	}
	sort.Slice(versions, func(i, j int) bool {
		if versions[i].StrategyID == versions[j].StrategyID {
			return strategybundle.CompareVersions(versions[i].Version, versions[j].Version) > 0
		}
		return versions[i].StrategyID < versions[j].StrategyID
	})
	return map[string]any{"versions": versions}, nil
}

func (server *Server) handleReadSiteFacts(ctx context.Context, _ map[string]any) (map[string]any, error) {
	if server.cfg.Truth == nil {
		return nil, fmt.Errorf("truth service is nil")
	}
	facts, err := server.cfg.Truth.SiteFacts(ctx)
	if err != nil {
		return nil, err
	}
	return structToObject(facts)
}

func (server *Server) handleReadOperatorPolicy(ctx context.Context, _ map[string]any) (map[string]any, error) {
	if server.cfg.Truth == nil {
		return nil, fmt.Errorf("truth service is nil")
	}
	policy, err := server.cfg.Truth.OperatorPolicy(ctx)
	if err != nil {
		return nil, err
	}
	return structToObject(policy)
}

func (server *Server) handleReadLeaders(ctx context.Context, _ map[string]any) (map[string]any, error) {
	if server.cfg.Truth == nil {
		return nil, fmt.Errorf("truth service is nil")
	}
	leaders, err := server.cfg.Truth.Leaders(ctx)
	if err != nil {
		return nil, err
	}
	return structToObject(leaders)
}

func (server *Server) handleReadLeaderScore(ctx context.Context, args map[string]any) (map[string]any, error) {
	if server.cfg.Truth == nil {
		return nil, fmt.Errorf("truth service is nil")
	}
	address, err := requiredStringArg(args, "address")
	if err != nil {
		return nil, err
	}
	leader, err := server.cfg.Truth.LeaderScore(ctx, address)
	if err != nil {
		return nil, err
	}
	return structToObject(leader)
}

func (server *Server) handleWriteBacktestRun(ctx context.Context, args map[string]any) (map[string]any, error) {
	if err := server.authorize(args); err != nil {
		return nil, err
	}
	if server.cfg.Backtests == nil {
		return nil, fmt.Errorf("backtest runner is nil")
	}
	configPath, err := requiredStringArg(args, "config_path")
	if err != nil {
		return nil, err
	}
	refresh, err := boolArg(args, "refresh", false)
	if err != nil {
		return nil, err
	}
	result, err := server.cfg.Backtests.RunConfig(ctx, configPath, refresh)
	if err != nil {
		return nil, err
	}
	return structToObject(result)
}

func (server *Server) handleWritePromotionRequest(ctx context.Context, args map[string]any) (map[string]any, error) {
	if err := server.authorize(args); err != nil {
		return nil, err
	}
	if server.cfg.Promotions == nil {
		return nil, fmt.Errorf("promotion manager is nil")
	}
	strategyID, err := requiredStringArg(args, "strategy_id")
	if err != nil {
		return nil, err
	}
	version, err := requiredStringArg(args, "version")
	if err != nil {
		return nil, err
	}
	configPath, err := requiredStringArg(args, "config_path")
	if err != nil {
		return nil, err
	}
	refresh, err := boolArg(args, "refresh", false)
	if err != nil {
		return nil, err
	}
	result, err := server.cfg.Promotions.CreateRequest(ctx, promotion.CreateInput{
		StrategyID: strategyID,
		Version:    version,
		ConfigPath: configPath,
		Refresh:    refresh,
	})
	if err != nil {
		return nil, err
	}
	return structToObject(result)
}

func (server *Server) handleWriteLiveFlatten(ctx context.Context, args map[string]any) (map[string]any, error) {
	if err := server.authorize(args); err != nil {
		return nil, err
	}
	if server.cfg.Live == nil {
		return nil, fmt.Errorf("live ops is nil")
	}
	symbol, err := requiredStringArg(args, "symbol")
	if err != nil {
		return nil, err
	}
	result, err := server.cfg.Live.FlattenSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return structToObject(result)
}

func (server *Server) handleWriteLeaderScore(ctx context.Context, args map[string]any) (map[string]any, error) {
	if err := server.authorize(args); err != nil {
		return nil, err
	}
	if server.cfg.Truth == nil {
		return nil, fmt.Errorf("truth service is nil")
	}
	address, err := requiredStringArg(args, "address")
	if err != nil {
		return nil, err
	}
	total, err := floatArg(args, "total")
	if err != nil {
		return nil, err
	}
	label, _ := optionalStringArg(args, "label")
	status, _ := optionalStringArg(args, "status")
	grade, _ := optionalStringArg(args, "grade")
	note, _ := optionalStringArg(args, "note")
	components, err := componentsArg(args, "components")
	if err != nil {
		return nil, err
	}
	leader, err := server.cfg.Truth.UpsertLeaderScore(ctx, truth.LeaderScoreInput{
		Address:    address,
		Label:      label,
		Status:     status,
		Total:      total,
		Grade:      grade,
		Note:       note,
		Components: components,
	})
	if err != nil {
		return nil, err
	}
	return structToObject(leader)
}

func (server *Server) authorize(args map[string]any) error {
	if server.cfg.WriteAuthToken == "" {
		return fmt.Errorf("write tools are disabled")
	}
	token, err := requiredStringArg(args, "auth_token")
	if err != nil {
		return err
	}
	if token != server.cfg.WriteAuthToken {
		return fmt.Errorf("write auth denied")
	}
	return nil
}

func (server *Server) protocolVersion(requested string) string {
	switch requested {
	case ProtocolVersionLatest, protocolVersionCompat:
		return requested
	default:
		return ProtocolVersionLatest
	}
}

func (server *Server) errorResponse(id json.RawMessage, code int, message string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}

func toolSuccessResult(payload map[string]any) map[string]any {
	return map[string]any{
		"content":           []map[string]any{{"type": "text", "text": mustJSONString(payload)}},
		"structuredContent": payload,
		"isError":           false,
	}
}

func toolErrorResult(err error) map[string]any {
	message := err.Error()
	return map[string]any{
		"content":           []map[string]any{{"type": "text", "text": message}},
		"structuredContent": map[string]any{"error": message},
		"isError":           true,
	}
}

func mustJSONString(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(body)
}

func structToObject(value any) (map[string]any, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func requiredStringArg(args map[string]any, key string) (string, error) {
	value, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return text, nil
}

func optionalStringArg(args map[string]any, key string) (string, bool) {
	value, ok := args[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	return text, true
}

func boolArg(args map[string]any, key string, fallback bool) (bool, error) {
	value, ok := args[key]
	if !ok {
		return fallback, nil
	}
	flag, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be boolean", key)
	}
	return flag, nil
}

func intArg(args map[string]any, key string, fallback int) (int, error) {
	value, ok := args[key]
	if !ok {
		return fallback, nil
	}
	number, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("%s must be integer", key)
	}
	return int(number), nil
}

func floatArg(args map[string]any, key string) (float64, error) {
	value, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}
	number, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("%s must be number", key)
	}
	return number, nil
}

func componentsArg(args map[string]any, key string) (map[string]float64, error) {
	value, ok := args[key]
	if !ok {
		return nil, nil
	}
	items, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be object", key)
	}
	out := make(map[string]float64, len(items))
	for name, raw := range items {
		number, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("%s.%s must be number", key, name)
		}
		out[name] = number
	}
	return out, nil
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema() map[string]any { return map[string]any{"type": "string"} }
func boolSchema() map[string]any   { return map[string]any{"type": "boolean"} }
