package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func (service *Service) sendTelegram(ctx context.Context, text string) error {
	return service.SendTelegramMessage(ctx, service.cfg.TelegramAlertChat, text)
}

func (service *Service) SendTelegramMessage(ctx context.Context, chatID, text string) error {
	if service.cfg.TelegramBotToken == "" || chatID == "" {
		return fmt.Errorf("telegram credentials are incomplete")
	}
	payload := map[string]string{
		"chat_id": chatID,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, service.baseURL+"/bot"+service.cfg.TelegramBotToken+"/sendMessage", bytes.NewReader(body))
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
		return fmt.Errorf("telegram status=%d", response.StatusCode)
	}
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := decodeJSON(response.Body, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("telegram api error: %s", result.Description)
	}
	return nil
}

func (service *Service) ListTelegramUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]telegramCommandUpdate, error) {
	if service.cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("telegram credentials are incomplete")
	}
	query := url.Values{}
	if offset > 0 {
		query.Set("offset", strconv.FormatInt(offset, 10))
	}
	if timeout > 0 {
		query.Set("timeout", strconv.Itoa(int(timeout.Seconds())))
	}
	target := service.baseURL + "/bot" + service.cfg.TelegramBotToken + "/getUpdates"
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	response, err := service.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("telegram status=%d", response.StatusCode)
	}
	var payload struct {
		OK          bool                    `json:"ok"`
		Description string                  `json:"description"`
		Result      []telegramCommandUpdate `json:"result"`
	}
	if err := decodeJSON(response.Body, &payload); err != nil {
		return nil, err
	}
	if !payload.OK {
		return nil, fmt.Errorf("telegram api error: %s", payload.Description)
	}
	return payload.Result, nil
}
