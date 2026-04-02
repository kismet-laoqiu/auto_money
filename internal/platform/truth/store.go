package truth

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type StoreConfig struct {
	SiteFactsPath      string
	OperatorPolicyPath string
	LeadersPath        string
	Now                func() time.Time
}

type Store struct {
	cfg StoreConfig
}

type SiteFacts struct {
	RepoRoot      string             `yaml:"repo_root" json:"repo_root"`
	PrimaryBranch string             `yaml:"primary_branch" json:"primary_branch"`
	Platformd     SiteFactsPlatformd `yaml:"platformd" json:"platformd"`
	Notifierd     SiteFactsNotifierd `yaml:"notifierd" json:"notifierd"`
	Runtime       SiteFactsRuntime   `yaml:"runtime" json:"runtime"`
	ControlPlane  ControlPlaneFacts  `yaml:"control_plane" json:"control_plane"`
	RiskDefaults  RiskDefaults       `yaml:"risk_defaults" json:"risk_defaults"`
}

type SiteFactsPlatformd struct {
	PublicBaseURL string `yaml:"public_base_url" json:"public_base_url"`
	LocalBaseURL  string `yaml:"local_base_url" json:"local_base_url"`
}

type SiteFactsNotifierd struct {
	LocalHealthURL string `yaml:"local_health_url" json:"local_health_url"`
}

type SiteFactsRuntime struct {
	StateDB         string `yaml:"state_db" json:"state_db"`
	WarehouseConfig string `yaml:"warehouse_config" json:"warehouse_config"`
}

type ControlPlaneFacts struct {
	TelegramInboundOwner   string `yaml:"telegram_inbound_owner" json:"telegram_inbound_owner"`
	OpenClawTelegramEnable bool   `yaml:"openclaw_telegram_enabled" json:"openclaw_telegram_enabled"`
	OpenClawDingTalkEnable bool   `yaml:"openclaw_dingtalk_enabled" json:"openclaw_dingtalk_enabled"`
}

type RiskDefaults struct {
	Leverage              int     `yaml:"leverage" json:"leverage"`
	MaxOrderNotionalUSDT  float64 `yaml:"max_order_notional_usdt" json:"max_order_notional_usdt"`
	MaxTotalExposureUSDT  float64 `yaml:"max_total_exposure_usdt" json:"max_total_exposure_usdt"`
	MaxConcurrentPosition int     `yaml:"max_concurrent_positions" json:"max_concurrent_positions"`
	DailyMaxLossUSDT      float64 `yaml:"daily_max_loss_usdt" json:"daily_max_loss_usdt"`
}

type OperatorPolicy struct {
	WriteActionsRequireConfirmation []string       `yaml:"write_actions_require_confirmation" json:"write_actions_require_confirmation"`
	Channels                        PolicyChannels `yaml:"channels" json:"channels"`
	DefaultLanguage                 string         `yaml:"default_language" json:"default_language"`
	Timezone                        string         `yaml:"timezone" json:"timezone"`
}

type PolicyChannels struct {
	Web      PolicyWeb      `yaml:"web" json:"web"`
	Telegram PolicyTelegram `yaml:"telegram" json:"telegram"`
	Dingtalk PolicyDingTalk `yaml:"dingtalk" json:"dingtalk"`
}

type PolicyWeb struct {
	PublicReadOnly      bool `yaml:"public_read_only" json:"public_read_only"`
	PrivateWriteRequired bool `yaml:"private_write_required" json:"private_write_required"`
}

type PolicyTelegram struct {
	Mode string `yaml:"mode" json:"mode"`
}

type PolicyDingTalk struct {
	Mode string `yaml:"mode" json:"mode"`
}

type LeadersFile struct {
	Hyperliquid []Leader `yaml:"hyperliquid" json:"hyperliquid"`
}

type Leader struct {
	Address      string      `yaml:"address" json:"address"`
	Label        string      `yaml:"label,omitempty" json:"label,omitempty"`
	Status       string      `yaml:"status,omitempty" json:"status,omitempty"`
	LastScoredAt string      `yaml:"last_scored_at,omitempty" json:"last_scored_at,omitempty"`
	Score        LeaderScore `yaml:"score" json:"score"`
}

type LeaderScore struct {
	Total      float64            `yaml:"total" json:"total"`
	Grade      string             `yaml:"grade,omitempty" json:"grade,omitempty"`
	Note       string             `yaml:"note,omitempty" json:"note,omitempty"`
	Components map[string]float64 `yaml:"components,omitempty" json:"components,omitempty"`
}

type LeaderScoreInput struct {
	Address    string
	Label      string
	Status     string
	Total      float64
	Grade      string
	Note       string
	Components map[string]float64
}

func NewStore(cfg StoreConfig) *Store {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Store{cfg: cfg}
}

func (store *Store) SiteFacts(context.Context) (SiteFacts, error) {
	return store.LoadSiteFacts()
}

func (store *Store) OperatorPolicy(context.Context) (OperatorPolicy, error) {
	return store.LoadOperatorPolicy()
}

func (store *Store) Leaders(context.Context) (LeadersFile, error) {
	return store.LoadLeaders()
}

func (store *Store) LeaderScore(_ context.Context, address string) (Leader, error) {
	leaders, err := store.LoadLeaders()
	if err != nil {
		return Leader{}, err
	}
	needle := normalizeAddress(address)
	for _, leader := range leaders.Hyperliquid {
		if normalizeAddress(leader.Address) == needle {
			return leader, nil
		}
	}
	return Leader{}, fmt.Errorf("leader %s not found", address)
}

func (store *Store) UpsertLeaderScore(_ context.Context, input LeaderScoreInput) (Leader, error) {
	if strings.TrimSpace(input.Address) == "" {
		return Leader{}, fmt.Errorf("address is required")
	}
	leaders, err := store.LoadLeaders()
	if err != nil {
		if os.IsNotExist(err) {
			leaders = LeadersFile{}
		} else {
			return Leader{}, err
		}
	}
	needle := normalizeAddress(input.Address)
	leader := Leader{
		Address:      needle,
		Label:        strings.TrimSpace(input.Label),
		Status:       strings.TrimSpace(input.Status),
		LastScoredAt: store.cfg.Now().UTC().Format(time.RFC3339),
		Score: LeaderScore{
			Total:      input.Total,
			Grade:      strings.TrimSpace(input.Grade),
			Note:       strings.TrimSpace(input.Note),
			Components: cloneComponents(input.Components),
		},
	}
	index := -1
	for i, item := range leaders.Hyperliquid {
		if normalizeAddress(item.Address) == needle {
			index = i
			if leader.Label == "" {
				leader.Label = item.Label
			}
			if leader.Status == "" {
				leader.Status = item.Status
			}
			break
		}
	}
	if index >= 0 {
		leaders.Hyperliquid[index] = leader
	} else {
		leaders.Hyperliquid = append(leaders.Hyperliquid, leader)
	}
	sort.Slice(leaders.Hyperliquid, func(i, j int) bool {
		return leaders.Hyperliquid[i].Address < leaders.Hyperliquid[j].Address
	})
	body, err := yaml.Marshal(&leaders)
	if err != nil {
		return Leader{}, err
	}
	if err := os.WriteFile(store.cfg.LeadersPath, body, 0o644); err != nil {
		return Leader{}, fmt.Errorf("write leaders %s: %w", store.cfg.LeadersPath, err)
	}
	return leader, nil
}

func (store *Store) LoadSiteFacts() (SiteFacts, error) {
	var facts SiteFacts
	if err := loadYAML(store.cfg.SiteFactsPath, &facts); err != nil {
		return SiteFacts{}, err
	}
	return facts, nil
}

func (store *Store) LoadOperatorPolicy() (OperatorPolicy, error) {
	var policy OperatorPolicy
	if err := loadYAML(store.cfg.OperatorPolicyPath, &policy); err != nil {
		return OperatorPolicy{}, err
	}
	if policy.DefaultLanguage == "" {
		policy.DefaultLanguage = "zh-CN"
	}
	if policy.Timezone == "" {
		policy.Timezone = "Asia/Shanghai"
	}
	return policy, nil
}

func (store *Store) LoadLeaders() (LeadersFile, error) {
	var leaders LeadersFile
	if err := loadYAML(store.cfg.LeadersPath, &leaders); err != nil {
		return LeadersFile{}, err
	}
	for i := range leaders.Hyperliquid {
		leaders.Hyperliquid[i].Address = normalizeAddress(leaders.Hyperliquid[i].Address)
	}
	sort.Slice(leaders.Hyperliquid, func(i, j int) bool {
		return leaders.Hyperliquid[i].Address < leaders.Hyperliquid[j].Address
	})
	return leaders, nil
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

func normalizeAddress(address string) string {
	return strings.ToLower(strings.TrimSpace(address))
}

func cloneComponents(input map[string]float64) map[string]float64 {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]float64, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
