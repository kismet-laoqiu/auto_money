package strategybundle

import (
	"fmt"
	"os"
	"path/filepath"
)

func Validate(root string) error {
	for _, name := range []string{"strategy.yaml", "universe.yaml", "risk.yaml", "score.cel", "gates.cel"} {
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("bundle missing required file %s", name)
			}
			return fmt.Errorf("stat %s: %w", path, err)
		}
		if info.IsDir() {
			return fmt.Errorf("bundle file %s is a directory", name)
		}
	}
	bundle, err := LoadUnchecked(root)
	if err != nil {
		return err
	}
	if bundle.StrategyID == "" {
		return fmt.Errorf("bundle strategy_id is required")
	}
	if bundle.Version == "" {
		return fmt.Errorf("bundle version is required")
	}
	if bundle.Strategy.FastSMA <= 0 || bundle.Strategy.SlowSMA <= 0 {
		return fmt.Errorf("bundle strategy SMA windows must be positive")
	}
	if len(bundle.Universe.Datasets) == 0 && len(bundle.Universe.Symbols) == 0 {
		return fmt.Errorf("bundle universe must declare datasets or symbols")
	}
	if bundle.Risk.MaxLeverage <= 0 {
		return fmt.Errorf("bundle risk.max_leverage must be positive")
	}
	if bundle.ScoreCEL == "" {
		return fmt.Errorf("bundle score.cel must not be empty")
	}
	if bundle.GatesCEL == "" {
		return fmt.Errorf("bundle gates.cel must not be empty")
	}
	return nil
}

func LoadUnchecked(root string) (Bundle, error) {
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
