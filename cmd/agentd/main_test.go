package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	platformjobs "quantlab/internal/platform/jobs"
)

type stubResearchRunner struct {
	request platformjobs.Request
	result  platformjobs.Result
	err     error
}

func (runner *stubResearchRunner) Run(_ context.Context, request platformjobs.Request) (platformjobs.Result, error) {
	runner.request = request
	return runner.result, runner.err
}

func TestRunCompatWithRunnerUsesResearchJob(t *testing.T) {
	configPath := filepath.Join("..", "..", "configs", "demo-mstr-bundle.yaml")
	runner := &stubResearchRunner{result: platformjobs.Result{
		GeneratedAt: time.Unix(1710000000, 0).UTC(),
		Kind:        platformjobs.KindNightlyReport,
		StrategyID:  "mstr-wave-fib",
		RequestID:   "resp-1",
		Status:      "completed",
	}}
	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer readPipe.Close()
	defer writePipe.Close()
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()

	if err := runCompatWithRunner([]string{"-config", configPath, "-kind", "nightly_report", "-datasets", "MSTRUSDT:1h,BTCUSDT:1h"}, runner); err != nil {
		t.Fatalf("run compat with runner: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if runner.request.Kind != platformjobs.KindNightlyReport {
		t.Fatalf("unexpected kind: %+v", runner.request)
	}
	if runner.request.StrategyID != "mstr-wave-fib" || runner.request.ConfigPath != configPath {
		t.Fatalf("compat request did not inherit config bundle: %+v", runner.request)
	}
	if len(runner.request.Datasets) != 2 || runner.request.Datasets[0] != "MSTRUSDT:1h" || runner.request.Datasets[1] != "BTCUSDT:1h" {
		t.Fatalf("unexpected datasets: %+v", runner.request.Datasets)
	}
	if !strings.Contains(string(output), `"request_id": "resp-1"`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunCompatWithRunnerDefaultsToNightlyReport(t *testing.T) {
	configPath := filepath.Join("..", "..", "configs", "demo-mstr-bundle.yaml")
	runner := &stubResearchRunner{result: platformjobs.Result{RequestID: "resp-2", Status: "completed"}}
	if err := runCompatWithRunner([]string{"-config", configPath}, runner); err != nil {
		t.Fatalf("run compat with default kind: %v", err)
	}
	if runner.request.Kind != platformjobs.KindNightlyReport {
		t.Fatalf("expected nightly_report default, got %+v", runner.request)
	}
	if runner.request.StrategyID != "mstr-wave-fib" {
		t.Fatalf("expected strategy from bundle, got %+v", runner.request)
	}
}
