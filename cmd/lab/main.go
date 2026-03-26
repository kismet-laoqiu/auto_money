package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"quantlab/internal/adapters"
	"quantlab/internal/config"
	"quantlab/internal/core"
)

type evalOutput struct {
	GeneratedAt    time.Time          `json:"generated_at"`
	MetricName     string             `json:"metric_name"`
	ObjectiveScore float64            `json:"objective_score"`
	Aggregate      map[string]float64 `json:"aggregate"`
	Reports        []core.Report      `json:"reports"`
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
	fmt.Fprintln(os.Stderr, "usage: lab <fetch|eval> [flags]")
}

func runFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	configPath := fs.String("config", "configs/baseline.yaml", "config file")
	refresh := fs.Bool("refresh", true, "refresh remote data")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
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
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	configPath := fs.String("config", "configs/baseline.yaml", "config file")
	refresh := fs.Bool("refresh", false, "refresh remote data")
	pretty := fs.Bool("pretty", true, "pretty print json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	ctx := context.Background()
	client := adapters.NewClient()
	reports := make([]core.Report, 0, len(cfg.Datasets))
	for _, spec := range cfg.Datasets {
		dataset, err := client.EnsureDataset(ctx, cfg.CacheDir, spec, *refresh)
		if err != nil {
			return err
		}
		report := core.BacktestDataset(dataset, cfg.Strategy, cfg.Objective)
		if err := adapters.WriteReportArtifacts(cfg.ArtifactDir, report); err != nil {
			return err
		}
		reports = append(reports, report)
	}

	aggregate := core.AggregateReports(reports)
	output := evalOutput{
		GeneratedAt:    time.Now().UTC(),
		MetricName:     "objective_score",
		ObjectiveScore: aggregate.ObjectiveScore,
		Aggregate:      aggregate.Metrics,
		Reports:        reports,
	}
	encoder := json.NewEncoder(os.Stdout)
	if *pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(output)
}
