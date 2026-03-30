package strategybundle

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type VersionInfo struct {
	StrategyID  string
	Version     string
	RootPath    string
	Description string
	UpdatedAt   time.Time
}

type Registry struct {
	Root string
}

func NewRegistry(root string) Registry {
	return Registry{Root: filepath.Clean(root)}
}

func (registry Registry) List(strategyID string) ([]VersionInfo, error) {
	if registry.Root == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(registry.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	versions := make([]VersionInfo, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strategyID != "" && entry.Name() != strategyID {
			continue
		}
		versionsDir := filepath.Join(registry.Root, entry.Name(), "versions")
		versionEntries, err := os.ReadDir(versionsDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, versionEntry := range versionEntries {
			if !versionEntry.IsDir() {
				continue
			}
			rootPath := filepath.Join(versionsDir, versionEntry.Name())
			bundle, err := Load(rootPath)
			if err != nil {
				return nil, err
			}
			updatedAt := time.Time{}
			if info, err := os.Stat(filepath.Join(rootPath, "strategy.yaml")); err == nil {
				updatedAt = info.ModTime().UTC()
			}
			versions = append(versions, VersionInfo{
				StrategyID:  bundle.StrategyID,
				Version:     bundle.Version,
				RootPath:    rootPath,
				Description: bundle.Description,
				UpdatedAt:   updatedAt,
			})
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		if versions[i].StrategyID == versions[j].StrategyID {
			return CompareVersions(versions[i].Version, versions[j].Version) > 0
		}
		return versions[i].StrategyID < versions[j].StrategyID
	})
	return versions, nil
}

func (registry Registry) Get(strategyID, version string) (Bundle, error) {
	return Load(filepath.Join(registry.Root, strategyID, "versions", version))
}

func CompareVersions(left, right string) int {
	leftParts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(left, "v"), "V"), ".")
	rightParts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(right, "v"), "V"), ".")
	maxParts := len(leftParts)
	if len(rightParts) > maxParts {
		maxParts = len(rightParts)
	}
	for i := 0; i < maxParts; i++ {
		leftPart := versionPart(leftParts, i)
		rightPart := versionPart(rightParts, i)
		leftValue, leftErr := strconv.Atoi(leftPart)
		rightValue, rightErr := strconv.Atoi(rightPart)
		switch {
		case leftErr == nil && rightErr == nil:
			if leftValue < rightValue {
				return -1
			}
			if leftValue > rightValue {
				return 1
			}
		default:
			if leftPart < rightPart {
				return -1
			}
			if leftPart > rightPart {
				return 1
			}
		}
	}
	return 0
}

func versionPart(parts []string, index int) string {
	if index >= len(parts) {
		return "0"
	}
	return parts[index]
}
