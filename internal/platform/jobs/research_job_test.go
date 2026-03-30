package jobs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quantlab/internal/agent"
)

func TestResearchJobRunWritesArtifactsAndToolCallSummary(t *testing.T) {
	service := &fakeResearchService{
		startResponse: agent.CreateResponse{ID: "resp-1", Status: "in_progress", Background: true},
		pollResponses: []agent.CreateResponse{
			{ID: "resp-1", Status: "in_progress", Background: true},
			{
				ID:     "resp-1",
				Status: "completed",
				Output: []agent.OutputItem{
					{Type: "web_search_call"},
					{Type: "message", Content: []agent.ContentPart{{Type: "output_text", Text: "nightly summary ready"}}},
				},
			},
		},
	}
	job := NewResearchJob(Config{
		Service: service,
		Now: func() time.Time {
			return time.Date(2026, 3, 29, 15, 0, 0, 0, time.UTC)
		},
	})

	result, err := job.Run(context.Background(), Request{
		Kind:         KindNightlyReport,
		StrategyID:   "mstr-wave-fib",
		ArtifactRoot: t.TempDir(),
		PollInterval: time.Millisecond,
		Timeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("run research job: %v", err)
	}
	if result.RequestID != "resp-1" || result.Status != "completed" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Type != "web_search_call" || result.ToolCalls[0].Count != 1 {
		t.Fatalf("unexpected tool calls: %+v", result.ToolCalls)
	}
	if service.startCalls != 1 || service.pollCalls != 2 {
		t.Fatalf("unexpected service calls: %+v", service)
	}
	outputBody, err := os.ReadFile(result.OutputArtifact)
	if err != nil {
		t.Fatalf("read output artifact: %v", err)
	}
	if !strings.Contains(string(outputBody), "nightly summary ready") {
		t.Fatalf("unexpected output artifact: %s", outputBody)
	}
	summaryBody, err := os.ReadFile(filepath.Join(result.ArtifactDir, "summary.json"))
	if err != nil {
		t.Fatalf("read summary artifact: %v", err)
	}
	var summary Result
	if err := json.Unmarshal(summaryBody, &summary); err != nil {
		t.Fatalf("decode summary artifact: %v", err)
	}
	if summary.RequestID != "resp-1" || summary.Status != "completed" {
		t.Fatalf("unexpected summary artifact: %+v", summary)
	}
	responseBody, err := os.ReadFile(filepath.Join(result.ArtifactDir, "response.json"))
	if err != nil {
		t.Fatalf("read response artifact: %v", err)
	}
	if !json.Valid(responseBody) {
		t.Fatalf("response artifact must be json: %s", responseBody)
	}
}

func TestResearchJobRunBuildsDefaultRequestsForSupportedKinds(t *testing.T) {
	tests := []struct {
		name    string
		request Request
		want    string
	}{
		{
			name: "theory study",
			request: Request{
				Kind:         KindTheoryStudy,
				StrategyID:   "mstr-wave-fib",
				ArtifactRoot: t.TempDir(),
			},
			want: "theory",
		},
		{
			name: "dataset scout",
			request: Request{
				Kind:         KindDatasetScout,
				StrategyID:   "mstr-wave-fib",
				Datasets:     []string{"BTCUSDT:1h", "MSTRUSDT:1h"},
				ArtifactRoot: t.TempDir(),
			},
			want: "dataset",
		},
		{
			name: "backtest diff summary",
			request: Request{
				Kind:           KindBacktestDiffSummary,
				StrategyID:     "mstr-wave-fib",
				BaselineRunID:  "run-baseline",
				CandidateRunID: "run-candidate",
				ArtifactRoot:   t.TempDir(),
			},
			want: "backtest",
		},
		{
			name: "nightly report",
			request: Request{
				Kind:         KindNightlyReport,
				StrategyID:   "mstr-wave-fib",
				ArtifactRoot: t.TempDir(),
			},
			want: "nightly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeResearchService{
				startResponse: agent.CreateResponse{ID: "resp-1", Status: "completed", Output: []agent.OutputItem{{Type: "message", Content: []agent.ContentPart{{Type: "output_text", Text: "ok"}}}}},
			}
			job := NewResearchJob(Config{Service: service})
			result, err := job.Run(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("run research job: %v", err)
			}
			if result.Status != "completed" {
				t.Fatalf("unexpected result: %+v", result)
			}
			if service.lastJob.Kind != string(tt.request.Kind) {
				t.Fatalf("unexpected job kind: %+v", service.lastJob)
			}
			if service.lastJob.Subject == "" {
				t.Fatalf("expected non-empty subject: %+v", service.lastJob)
			}
			if !strings.Contains(strings.ToLower(service.lastJob.Body), tt.want) {
				t.Fatalf("expected %q in body: %+v", tt.want, service.lastJob)
			}
		})
	}
}

func TestResearchJobRunRejectsUnsupportedKind(t *testing.T) {
	job := NewResearchJob(Config{Service: &fakeResearchService{}})
	_, err := job.Run(context.Background(), Request{Kind: Kind("unsupported"), ArtifactRoot: t.TempDir()})
	if err == nil {
		t.Fatalf("expected unsupported kind error")
	}
}

type fakeResearchService struct {
	startResponse agent.CreateResponse
	pollResponses []agent.CreateResponse
	lastJob       agent.JobRequest
	startCalls    int
	pollCalls     int
}

func (service *fakeResearchService) StartBackgroundJob(_ context.Context, job agent.JobRequest) (agent.CreateResponse, error) {
	service.startCalls++
	service.lastJob = job
	return service.startResponse, nil
}

func (service *fakeResearchService) PollResponse(_ context.Context, _ string) (agent.CreateResponse, error) {
	service.pollCalls++
	if len(service.pollResponses) == 0 {
		return agent.CreateResponse{ID: "resp-1", Status: "completed"}, nil
	}
	response := service.pollResponses[0]
	service.pollResponses = service.pollResponses[1:]
	return response, nil
}
