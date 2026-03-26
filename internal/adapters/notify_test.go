package adapters

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"quantlab/internal/config"
)

func TestNewNotifierReturnsStdoutByDefault(t *testing.T) {
	notifier := NewNotifier(config.NotifyConfig{})
	if _, ok := notifier.(StdoutNotifier); !ok {
		t.Fatalf("expected stdout notifier, got %T", notifier)
	}
}

func TestDingTalkNotifierNotifySignsAndPrefixesMessage(t *testing.T) {
	var gotContent string
	var handlerErr string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		timestamp := request.URL.Query().Get("timestamp")
		sign := request.URL.Query().Get("sign")
		if timestamp == "" || sign == "" {
			handlerErr = fmt.Sprintf("expected signed webhook query, got %s", request.URL.RawQuery)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		mac := hmac.New(sha256.New, []byte("demo-secret"))
		mac.Write([]byte(timestamp + "\n" + "demo-secret"))
		expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		if sign != expected {
			handlerErr = fmt.Sprintf("unexpected sign: got %s want %s", sign, expected)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			handlerErr = fmt.Sprintf("read body: %v", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		var payload struct {
			Text struct {
				Content string `json:"content"`
			} `json:"text"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			handlerErr = fmt.Sprintf("decode payload: %v", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		gotContent = payload.Text.Content
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := &DingTalkNotifier{
		webhook: server.URL,
		secret:  "demo-secret",
		keyword: "QUANT",
		client:  server.Client(),
	}
	if err := notifier.Notify(context.Background(), "entry signal"); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if handlerErr != "" {
		t.Fatal(handlerErr)
	}
	if gotContent != "QUANT\nentry signal" {
		t.Fatalf("unexpected content: %q", gotContent)
	}
}
