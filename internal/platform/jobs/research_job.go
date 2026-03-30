package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"quantlab/internal/agent"
)

type Service interface {
	StartBackgroundJob(ctx context.Context, job agent.JobRequest) (agent.CreateResponse, error)
	PollResponse(ctx context.Context, responseID string) (agent.CreateResponse, error)
}

type Config struct {
	Service Service
	Now     func() time.Time
}

type ResearchJob struct {
	cfg Config
}

type Kind string

const (
	KindTheoryStudy         Kind = "theory_study"
	KindDatasetScout        Kind = "dataset_scout"
	KindBacktestDiffSummary Kind = "backtest_diff_summary"
	KindNightlyReport       Kind = "nightly_report"
)

type Request struct {
	Kind           Kind          `json:"kind"`
	StrategyID     string        `json:"strategy_id,omitempty"`
	Subject        string        `json:"subject,omitempty"`
	Body           string        `json:"body,omitempty"`
	ConfigPath     string        `json:"config_path,omitempty"`
	BaselineRunID  string        `json:"baseline_run_id,omitempty"`
	CandidateRunID string        `json:"candidate_run_id,omitempty"`
	Datasets       []string      `json:"datasets,omitempty"`
	ArtifactRoot   string        `json:"artifact_root,omitempty"`
	PollInterval   time.Duration `json:"poll_interval,omitempty"`
	Timeout        time.Duration `json:"timeout,omitempty"`
}

type ToolCallSummary struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type Result struct {
	GeneratedAt      time.Time         `json:"generated_at"`
	RequestID        string            `json:"request_id"`
	Status           string            `json:"status"`
	Kind             Kind              `json:"kind"`
	StrategyID       string            `json:"strategy_id,omitempty"`
	Subject          string            `json:"subject"`
	Body             string            `json:"body"`
	ArtifactDir      string            `json:"artifact_dir"`
	RequestArtifact  string            `json:"request_artifact"`
	ResponseArtifact string            `json:"response_artifact"`
	OutputArtifact   string            `json:"output_artifact"`
	ToolCalls        []ToolCallSummary `json:"tool_calls,omitempty"`
	OutputText       string            `json:"output_text,omitempty"`
}

func NewResearchJob(cfg Config) *ResearchJob {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &ResearchJob{cfg: cfg}
}

func (job *ResearchJob) Run(ctx context.Context, request Request) (Result, error) {
	if job.cfg.Service == nil {
		return Result{}, fmt.Errorf("research service is nil")
	}
	normalized, err := normalizeRequest(request)
	if err != nil {
		return Result{}, err
	}
	generatedAt := job.cfg.Now().UTC()
	artifactDir := filepath.Join(normalized.ArtifactRoot, fmt.Sprintf("%s-%s-%s", generatedAt.Format("20060102T150405Z"), sanitizeName(string(normalized.Kind)), sanitizeName(firstNonEmpty(normalized.StrategyID, normalized.Subject))))
	result := Result{
		GeneratedAt:      generatedAt,
		Kind:             normalized.Kind,
		StrategyID:       normalized.StrategyID,
		Subject:          normalized.Subject,
		Body:             normalized.Body,
		ArtifactDir:      artifactDir,
		RequestArtifact:  filepath.Join(artifactDir, "request.json"),
		ResponseArtifact: filepath.Join(artifactDir, "response.json"),
		OutputArtifact:   filepath.Join(artifactDir, "output.txt"),
	}
	response, err := job.cfg.Service.StartBackgroundJob(ctx, agent.JobRequest{
		Kind:    string(normalized.Kind),
		Subject: normalized.Subject,
		Body:    normalized.Body,
	})
	if err != nil {
		return Result{}, err
	}
	response, waitErr := job.waitForCompletion(ctx, response, normalized.PollInterval, normalized.Timeout)
	result.RequestID = firstNonEmpty(response.ID, result.RequestID)
	result.Status = response.Status
	result.ToolCalls = summarizeToolCalls(response.Output)
	result.OutputText = strings.TrimSpace(response.OutputText())
	if err := writeArtifacts(result, normalized, response); err != nil {
		return result, err
	}
	return result, waitErr
}

func (job *ResearchJob) waitForCompletion(ctx context.Context, response agent.CreateResponse, pollInterval, timeout time.Duration) (agent.CreateResponse, error) {
	if isTerminalStatus(response.Status) {
		return response, nil
	}
	if response.ID == "" {
		return response, fmt.Errorf("research response id is empty")
	}
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	current := response
	for {
		select {
		case <-pollCtx.Done():
			return current, pollCtx.Err()
		case <-ticker.C:
			next, err := job.cfg.Service.PollResponse(pollCtx, response.ID)
			if err != nil {
				return current, err
			}
			current = next
			if isTerminalStatus(current.Status) {
				return current, nil
			}
		}
	}
}

func writeArtifacts(result Result, request Request, response agent.CreateResponse) error {
	if err := os.MkdirAll(result.ArtifactDir, 0o755); err != nil {
		return err
	}
	requestBody, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(result.RequestArtifact, append(requestBody, '\n'), 0o644); err != nil {
		return err
	}
	responseBody, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(result.ResponseArtifact, append(responseBody, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(result.OutputArtifact, []byte(result.OutputText+"\n"), 0o644); err != nil {
		return err
	}
	summaryPath := filepath.Join(result.ArtifactDir, "summary.json")
	summaryBody, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(summaryPath, append(summaryBody, '\n'), 0o644)
}

func normalizeRequest(request Request) (Request, error) {
	switch request.Kind {
	case KindTheoryStudy, KindDatasetScout, KindBacktestDiffSummary, KindNightlyReport:
	default:
		return Request{}, fmt.Errorf("unsupported research kind %q", request.Kind)
	}
	if request.ArtifactRoot == "" {
		request.ArtifactRoot = filepath.Join("artifacts", "platform", "research")
	}
	if request.PollInterval <= 0 {
		request.PollInterval = 2 * time.Second
	}
	if request.Timeout <= 0 {
		request.Timeout = 2 * time.Minute
	}
	if request.Subject == "" {
		request.Subject = defaultSubject(request)
	}
	if request.Body == "" {
		request.Body = defaultBody(request)
	}
	return request, nil
}

func defaultSubject(request Request) string {
	if request.StrategyID != "" {
		return request.StrategyID
	}
	return string(request.Kind)
}

func defaultBody(request Request) string {
	switch request.Kind {
	case KindTheoryStudy:
		return strings.TrimSpace(fmt.Sprintf("Study the trading theory behind strategy=%s. Focus on wave structure, fib confluence, volume confirmation, regime tags, support and resistance, then explain how the current bundle expresses those ideas.%s%s", firstNonEmpty(request.StrategyID, "unknown"), configPathClause(request.ConfigPath), datasetsClause(request.Datasets)))
	case KindDatasetScout:
		return strings.TrimSpace(fmt.Sprintf("Scout read-only dataset opportunities for strategy=%s. Identify useful symbols, intervals, coverage gaps, and data quality risks before any bundle change.%s%s", firstNonEmpty(request.StrategyID, "unknown"), datasetsClause(request.Datasets), configPathClause(request.ConfigPath)))
	case KindBacktestDiffSummary:
		return strings.TrimSpace(fmt.Sprintf("Compare backtest runs for strategy=%s. Baseline=%s candidate=%s. Summarize score delta, dataset hash delta, feature version delta, bundle version delta, artifact paths, and recommend the next action.%s", firstNonEmpty(request.StrategyID, "unknown"), firstNonEmpty(request.BaselineRunID, "unknown"), firstNonEmpty(request.CandidateRunID, "unknown"), configPathClause(request.ConfigPath)))
	case KindNightlyReport:
		return strings.TrimSpace(fmt.Sprintf("Prepare the nightly research report for strategy=%s. Summarize the latest backtest posture, promotion state, notable risks, dataset freshness, and the next operator actions.%s%s", firstNonEmpty(request.StrategyID, "platform"), configPathClause(request.ConfigPath), datasetsClause(request.Datasets)))
	default:
		return ""
	}
}

func datasetsClause(datasets []string) string {
	if len(datasets) == 0 {
		return ""
	}
	return " Datasets=" + strings.Join(datasets, ", ") + "."
}

func configPathClause(configPath string) string {
	if strings.TrimSpace(configPath) == "" {
		return ""
	}
	return " ConfigPath=" + configPath + "."
}

func summarizeToolCalls(items []agent.OutputItem) []ToolCallSummary {
	counts := make(map[string]int)
	order := make([]string, 0)
	for _, item := range items {
		kind := strings.ToLower(strings.TrimSpace(item.Type))
		if !isToolCallKind(kind) {
			continue
		}
		if counts[kind] == 0 {
			order = append(order, kind)
		}
		counts[kind]++
	}
	out := make([]ToolCallSummary, 0, len(order))
	for _, kind := range order {
		out = append(out, ToolCallSummary{Type: kind, Count: counts[kind]})
	}
	return out
}

func isToolCallKind(kind string) bool {
	return strings.Contains(kind, "_call") || kind == "function_call"
}

func isTerminalStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "failed", "cancelled", "canceled", "incomplete":
		return true
	default:
		return false
	}
}

func sanitizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "platform"
	}
	replacer := strings.NewReplacer("/", "-", " ", "-", ":", "-", ".", "-", "_", "-")
	value = replacer.Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
