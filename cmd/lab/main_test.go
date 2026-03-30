package main

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"quantlab/internal/backtest"
)

type stubEvalRunner struct {
	result     backtest.Result
	configPath string
	refresh    bool
}

func (runner *stubEvalRunner) RunConfig(_ context.Context, configPath string, refresh bool) (backtest.Result, error) {
	runner.configPath = configPath
	runner.refresh = refresh
	return runner.result, nil
}

func TestRunEvalWithRunnerUsesBacktestServiceOutput(t *testing.T) {
	runner := &stubEvalRunner{result: backtest.Result{
		GeneratedAt:    time.Unix(1710000000, 0).UTC(),
		MetricName:     "final_score",
		ObjectiveScore: 0.86,
		FinalScore:     0.91,
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

	if err := runEvalWithRunner([]string{"-config", "configs/demo-mstr-bundle.yaml", "-refresh"}, runner); err != nil {
		t.Fatalf("run eval with runner: %v", err)
	}
	_ = writePipe.Close()
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if runner.configPath != "configs/demo-mstr-bundle.yaml" || !runner.refresh {
		t.Fatalf("runner inputs not forwarded: %+v", runner)
	}
	if !strings.Contains(string(output), `"metric_name": "final_score"`) || !strings.Contains(string(output), `"final_score": 0.91`) {
		t.Fatalf("unexpected stdout: %s", output)
	}
}
