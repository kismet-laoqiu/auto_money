package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"quantlab/internal/agent"
	platformjobs "quantlab/internal/platform/jobs"
)

func main() {
	fs := flag.NewFlagSet("researchd", flag.ContinueOnError)
	kind := fs.String("kind", "", "research job kind")
	strategy := fs.String("strategy", "", "strategy id")
	subject := fs.String("subject", "", "job subject")
	body := fs.String("body", "", "job body override")
	configPath := fs.String("config-path", "", "related config path")
	baselineRunID := fs.String("baseline-run", "", "baseline backtest run id")
	candidateRunID := fs.String("candidate-run", "", "candidate backtest run id")
	datasets := fs.String("datasets", "", "comma separated dataset descriptors")
	artifactRoot := fs.String("artifact-root", "artifacts/platform/research", "artifact output root")
	pollInterval := fs.Duration("poll-interval", 2*time.Second, "poll interval")
	timeout := fs.Duration("timeout", 2*time.Minute, "overall timeout")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *kind == "" {
		fmt.Fprintln(os.Stderr, "kind is required")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	client := agent.NewHTTPResponsesClient(agent.HTTPClientConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	})
	service := agent.NewService(client, agent.Config{Store: true})
	job := platformjobs.NewResearchJob(platformjobs.Config{Service: service})
	result, err := job.Run(ctx, platformjobs.Request{
		Kind:           platformjobs.Kind(*kind),
		StrategyID:     *strategy,
		Subject:        *subject,
		Body:           *body,
		ConfigPath:     *configPath,
		BaselineRunID:  *baselineRunID,
		CandidateRunID: *candidateRunID,
		Datasets:       splitCSV(*datasets),
		ArtifactRoot:   *artifactRoot,
		PollInterval:   *pollInterval,
		Timeout:        *timeout,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
