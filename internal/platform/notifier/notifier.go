package notifier

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Channel string

const (
	ChannelTelegram Channel = "telegram"
	ChannelDingTalk Channel = "dingtalk"
)

type Kind string

const (
	KindFill              Kind = "fill"
	KindRiskHalt          Kind = "risk_halt"
	KindPromotionApproved Kind = "promotion_approved"
	KindCanaryDegraded    Kind = "canary_degraded"
	KindDailySummary      Kind = "daily_summary"
	KindDailySignal       Kind = "daily_signal"
	KindMarketAlert       Kind = "market_alert"
)

type Config struct {
	BaseURL           string
	DingTalkWebhook   string
	DingTalkSecret    string
	DingTalkKeyword   string
	TelegramBotToken  string
	TelegramAlertChat string
	HTTPClient        *http.Client
}

type Notification struct {
	Kind    Kind
	Title   string
	Summary string
	Details []string
}

type Service struct {
	baseURL string
	cfg     Config
	client  *http.Client
}

func New(cfg Config) *Service {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	return &Service{
		baseURL: baseURL,
		cfg:     cfg,
		client:  client,
	}
}

func (service *Service) Send(ctx context.Context, channel Channel, text string) error {
	switch channel {
	case ChannelTelegram:
		return service.sendTelegram(ctx, text)
	case ChannelDingTalk:
		return service.sendDingTalk(ctx, text)
	default:
		return fmt.Errorf("unsupported channel %q", channel)
	}
}

func (service *Service) Notify(ctx context.Context, notification Notification) error {
	for _, channel := range ChannelsForKind(notification.Kind) {
		if err := service.Send(ctx, channel, notification.Render()); err != nil {
			return err
		}
	}
	return nil
}

func ChannelsForKind(kind Kind) []Channel {
	switch kind {
	case KindFill, KindRiskHalt, KindPromotionApproved, KindCanaryDegraded, KindDailySummary:
		return []Channel{ChannelTelegram, ChannelDingTalk}
	case KindDailySignal, KindMarketAlert:
		return []Channel{ChannelDingTalk}
	default:
		return nil
	}
}

func (service *Service) sendDingTalk(ctx context.Context, text string) error {
	if service.cfg.DingTalkWebhook == "" {
		return fmt.Errorf("dingtalk webhook is empty")
	}
	webhook, err := service.resolveDingTalkWebhook(time.Now().UTC())
	if err != nil {
		return err
	}
	if service.cfg.DingTalkKeyword != "" && !strings.Contains(text, service.cfg.DingTalkKeyword) {
		text = service.cfg.DingTalkKeyword + "\n" + text
	}
	payload := map[string]any{
		"msgtype": "text",
		"text": map[string]string{
			"content": text,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := service.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("dingtalk status=%d", response.StatusCode)
	}
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := decodeJSON(response.Body, &result); err != nil {
		return err
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("dingtalk api error: %s", result.ErrMsg)
	}
	return nil
}

func (service *Service) resolveDingTalkWebhook(now time.Time) (string, error) {
	if service.cfg.DingTalkSecret == "" {
		return service.cfg.DingTalkWebhook, nil
	}
	parsed, err := url.Parse(service.cfg.DingTalkWebhook)
	if err != nil {
		return "", err
	}
	timestamp := fmt.Sprintf("%d", now.UnixMilli())
	mac := hmac.New(sha256.New, []byte(service.cfg.DingTalkSecret))
	mac.Write([]byte(timestamp + "\n" + service.cfg.DingTalkSecret))
	query := parsed.Query()
	query.Set("timestamp", timestamp)
	query.Set("sign", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func decodeJSON(reader io.Reader, target any) error {
	if err := json.NewDecoder(reader).Decode(target); err != nil && err != io.EOF {
		return err
	}
	return nil
}
