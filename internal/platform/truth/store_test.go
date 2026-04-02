package truth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreLoadsFactsPolicyAndLeaders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "site-facts.yaml"), `
repo_root: /repo
primary_branch: autoresearch/20260328-all-plan
platformd:
  public_base_url: http://47.250.138.143:8080
  local_base_url: http://127.0.0.1:8080
`)
	writeFile(t, filepath.Join(root, "operator-policy.yaml"), `
write_actions_require_confirmation:
  - flatten_symbol
channels:
  web:
    public_read_only: true
default_language: zh-CN
timezone: Asia/Shanghai
`)
	writeFile(t, filepath.Join(root, "leaders.yaml"), `
hyperliquid:
  - address: "0xabc"
    label: alpha
    score:
      total: 88
      grade: A
`)

	store := NewStore(StoreConfig{
		SiteFactsPath:      filepath.Join(root, "site-facts.yaml"),
		OperatorPolicyPath: filepath.Join(root, "operator-policy.yaml"),
		LeadersPath:        filepath.Join(root, "leaders.yaml"),
		Now:                func() time.Time { return time.Unix(1710000000, 0).UTC() },
	})

	facts, err := store.LoadSiteFacts()
	if err != nil {
		t.Fatalf("load site facts: %v", err)
	}
	if facts.RepoRoot != "/repo" || facts.PrimaryBranch != "autoresearch/20260328-all-plan" {
		t.Fatalf("unexpected facts: %+v", facts)
	}

	policy, err := store.LoadOperatorPolicy()
	if err != nil {
		t.Fatalf("load operator policy: %v", err)
	}
	if len(policy.WriteActionsRequireConfirmation) != 1 || !policy.Channels.Web.PublicReadOnly {
		t.Fatalf("unexpected policy: %+v", policy)
	}

	leaders, err := store.LoadLeaders()
	if err != nil {
		t.Fatalf("load leaders: %v", err)
	}
	if len(leaders.Hyperliquid) != 1 || leaders.Hyperliquid[0].Address != "0xabc" || leaders.Hyperliquid[0].Score.Total != 88 {
		t.Fatalf("unexpected leaders: %+v", leaders)
	}
}

func TestStoreUpsertLeaderScore(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "leaders.yaml"), "hyperliquid: []\n")

	store := NewStore(StoreConfig{
		LeadersPath: filepath.Join(root, "leaders.yaml"),
		Now:         func() time.Time { return time.Unix(1710000000, 0).UTC() },
	})

	leader, err := store.UpsertLeaderScore(context.Background(), LeaderScoreInput{
		Address: "0xabc",
		Label:   "alpha",
		Total:   91.5,
		Grade:   "A",
		Note:    "copyable",
		Components: map[string]float64{
			"consistency": 18,
			"drawdown":    14,
		},
	})
	if err != nil {
		t.Fatalf("upsert leader score: %v", err)
	}
	if leader.Address != "0xabc" || leader.Score.Total != 91.5 || leader.LastScoredAt == "" {
		t.Fatalf("unexpected leader: %+v", leader)
	}

	saved, err := store.LoadLeaders()
	if err != nil {
		t.Fatalf("reload leaders: %v", err)
	}
	if len(saved.Hyperliquid) != 1 || saved.Hyperliquid[0].Score.Components["consistency"] != 18 {
		t.Fatalf("unexpected saved leaders: %+v", saved)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
