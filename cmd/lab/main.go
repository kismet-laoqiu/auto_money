package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"quantlab/internal/adapters"
	"quantlab/internal/backtest"
	"quantlab/internal/replay"
	"quantlab/internal/strategybundle"
	"quantlab/internal/trader"
)

type evalRunner interface {
	RunConfig(ctx context.Context, configPath string, refresh bool) (backtest.Result, error)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "fetch":
		err = runFetch(os.Args[2:])
	case "eval":
		err = runEval(os.Args[2:])
	case "replay":
		err = runReplay(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: lab <fetch|eval|replay> [flags]")
}

func runFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	configPath := fs.String("config", "configs/baseline.yaml", "config file")
	refresh := fs.Bool("refresh", true, "refresh remote data")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, _, err := strategybundle.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	ctx := context.Background()
	client := adapters.NewClient()
	for _, spec := range cfg.Datasets {
		dataset, err := client.EnsureDataset(ctx, cfg.CacheDir, spec, *refresh)
		if err != nil {
			return err
		}
		fmt.Printf("fetched %s %s %s bars=%d\n", dataset.Provider, dataset.Symbol, dataset.Interval, len(dataset.Bars))
	}
	return nil
}

func runEval(args []string) error {
	return runEvalWithRunner(args, backtest.NewService(backtest.Config{Loader: adapters.NewClient()}))
}

func runEvalWithRunner(args []string, runner evalRunner) error {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	configPath := fs.String("config", "configs/baseline.yaml", "config file")
	refresh := fs.Bool("refresh", false, "refresh remote data")
	pretty := fs.Bool("pretty", true, "pretty print json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if runner == nil {
		return fmt.Errorf("backtest runner is nil")
	}

	result, err := runner.RunConfig(context.Background(), *configPath, *refresh)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	if *pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(result)
}

func runReplay(args []string) error {
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	configPath := fs.String("config", "configs/live.yaml", "config file")
	refresh := fs.Bool("refresh", false, "refresh remote data")
	pretty := fs.Bool("pretty", true, "pretty print json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, bundle, err := strategybundle.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	specs := replay.BuildReplaySpecs(cfg)
	events, err := replay.LoadEvents(context.Background(), cfg.CacheDir, specs, *refresh)
	if err != nil {
		return err
	}
	strategy := trader.Strategy(trader.LegacyRuleProfile{StrategyCfg: cfg.Strategy})
	if bundle != nil {
		strategy = trader.BundleStrategy{Bundle: *bundle}
	}
	engine := trader.NewEngine(trader.Config{
		ArmingState: trader.ArmingArmed,
		Strategy:    strategy,
	})
	harness := replay.NewHarness(engine, replay.NewLongOnlySimulator())
	report, err := harness.RunAndWrite(events, cfg.ArtifactDir)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	if *pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(report)
}
