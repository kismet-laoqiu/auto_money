package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"quantlab/internal/agent"
	platformjobs "quantlab/internal/platform/jobs"
	"quantlab/internal/strategybundle"
)

type researchRunner interface {
	Run(ctx context.Context, request platformjobs.Request) (platformjobs.Result, error)
}

func main() {
	if err := runCompat(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCompat(args []string) error {
	fmt.Fprintln(os.Stderr, "agentd compatibility shell: forwarding advisory jobs to researchd")
	client := agent.NewHTTPResponsesClient(agent.HTTPClientConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	})
	service := agent.NewService(client, agent.Config{Store: true})
	return runCompatWithRunner(args, platformjobs.NewResearchJob(platformjobs.Config{Service: service}))
}

func runCompatWithRunner(args []string, runner researchRunner) error {
	if runner == nil {
		return fmt.Errorf("research runner is nil")
	}
	fs := flag.NewFlagSet("agentd", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "configs/demo-bitget.yaml", "config file")
	kind := fs.String("kind", string(platformjobs.KindNightlyReport), "research job kind")
	strategyID := fs.String("strategy", "", "strategy id")
	subject := fs.String("subject", "", "job subject")
	body := fs.String("body", "", "job body override")
	baselineRunID := fs.String("baseline-run", "", "baseline backtest run id")
	candidateRunID := fs.String("candidate-run", "", "candidate backtest run id")
	datasets := fs.String("datasets", "", "comma separated dataset descriptors")
	artifactRoot := fs.String("artifact-root", "artifacts/platform/research", "artifact output root")
	pollInterval := fs.Duration("poll-interval", 2*time.Second, "poll interval")
	timeout := fs.Duration("timeout", 2*time.Minute, "overall timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_, bundle, err := strategybundle.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	resolvedStrategyID := strings.TrimSpace(*strategyID)
	if resolvedStrategyID == "" && bundle != nil {
		resolvedStrategyID = bundle.StrategyID
	}
	result, err := runner.Run(context.Background(), platformjobs.Request{
		Kind:           platformjobs.Kind(strings.TrimSpace(*kind)),
		StrategyID:     resolvedStrategyID,
		Subject:        strings.TrimSpace(*subject),
		Body:           strings.TrimSpace(*body),
		ConfigPath:     *configPath,
		BaselineRunID:  strings.TrimSpace(*baselineRunID),
		CandidateRunID: strings.TrimSpace(*candidateRunID),
		Datasets:       splitCSV(*datasets),
		ArtifactRoot:   *artifactRoot,
		PollInterval:   *pollInterval,
		Timeout:        *timeout,
	})
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
