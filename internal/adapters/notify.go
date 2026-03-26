package adapters

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"quantlab/internal/config"
)

type Notifier interface {
	Notify(context.Context, string) error
}

type StdoutNotifier struct{}

func (StdoutNotifier) Notify(_ context.Context, message string) error {
	fmt.Println(message)
	return nil
}

type DingTalkNotifier struct {
	webhook string
	secret  string
	keyword string
	client  *http.Client
}

func NewNotifier(cfg config.NotifyConfig) Notifier {
	if cfg.Enable && cfg.DingTalkWebhook != "" {
		return &DingTalkNotifier{
			webhook: cfg.DingTalkWebhook,
			secret:  cfg.DingTalkSecret,
			keyword: cfg.DingTalkKeyword,
			client:  &http.Client{Timeout: 10 * time.Second},
		}
	}
	return StdoutNotifier{}
}

func (notifier *DingTalkNotifier) Notify(ctx context.Context, message string) error {
	webhook, err := notifier.resolveWebhook(time.Now().UTC())
	if err != nil {
		return err
	}
	if notifier.keyword != "" && !strings.Contains(message, notifier.keyword) {
		message = notifier.keyword + "\n" + message
	}
	payload := map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": message},
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
	response, err := notifier.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("dingtalk status=%d", response.StatusCode)
	}
	return nil
}

func (notifier *DingTalkNotifier) resolveWebhook(now time.Time) (string, error) {
	if notifier.secret == "" {
		return notifier.webhook, nil
	}
	parsed, err := url.Parse(notifier.webhook)
	if err != nil {
		return "", err
	}
	timestamp := fmt.Sprintf("%d", now.UnixMilli())
	mac := hmac.New(sha256.New, []byte(notifier.secret))
	mac.Write([]byte(timestamp + "\n" + notifier.secret))
	query := parsed.Query()
	query.Set("timestamp", timestamp)
	query.Set("sign", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
