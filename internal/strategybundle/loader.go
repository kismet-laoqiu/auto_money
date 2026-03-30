package strategybundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"quantlab/internal/config"
)

type strategyDoc struct {
	StrategyID  string                 `yaml:"strategy_id"`
	Version     string                 `yaml:"version"`
	Description string                 `yaml:"description"`
	Strategy    config.StrategyConfig  `yaml:"strategy"`
	Objective   config.ObjectiveConfig `yaml:"objective"`
}

func Load(root string) (Bundle, error) {
	if err := Validate(root); err != nil {
		return Bundle{}, err
	}
	var strategy strategyDoc
	if err := loadYAML(filepath.Join(root, "strategy.yaml"), &strategy); err != nil {
		return Bundle{}, err
	}
	var universe Universe
	if err := loadYAML(filepath.Join(root, "universe.yaml"), &universe); err != nil {
		return Bundle{}, err
	}
	var risk Risk
	if err := loadYAML(filepath.Join(root, "risk.yaml"), &risk); err != nil {
		return Bundle{}, err
	}
	scoreCEL, err := readTrimmedFile(filepath.Join(root, "score.cel"))
	if err != nil {
		return Bundle{}, err
	}
	gatesCEL, err := readTrimmedFile(filepath.Join(root, "gates.cel"))
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{
		RootPath:    root,
		StrategyID:  strategy.StrategyID,
		Version:     strategy.Version,
		Description: strategy.Description,
		Strategy:    strategy.Strategy,
		Objective:   strategy.Objective,
		Universe:    universe,
		Risk:        risk,
		ScoreCEL:    scoreCEL,
		GatesCEL:    gatesCEL,
	}, nil
}

func loadYAML(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func readTrimmedFile(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return strings.TrimSpace(string(body)), nil
}
