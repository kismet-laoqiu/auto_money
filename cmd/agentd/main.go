package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"quantlab/internal/agent"
	"quantlab/internal/config"
)

func main() {
	fs := flag.NewFlagSet("agentd", flag.ContinueOnError)
	configPath := fs.String("config", "configs/demo-bitget.yaml", "config file")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !cfg.Live.Enabled {
		fmt.Fprintln(os.Stderr, "live.enabled=false")
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, cfg); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newService(cfg config.Config) (*agent.Service, error) {
	if !cfg.Live.Agent.AdvisoryOnly {
		return nil, fmt.Errorf("agentd requires live.agent.advisory_only=true")
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := agent.NewHTTPResponsesClient(agent.HTTPClientConfig{APIKey: apiKey})
	return agent.NewService(client, agent.Config{Store: true}), nil
}

func run(ctx context.Context, cfg config.Config) error {
	if !cfg.Live.Agent.Enabled {
		<-ctx.Done()
		return ctx.Err()
	}
	if _, err := newService(cfg); err != nil {
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}
