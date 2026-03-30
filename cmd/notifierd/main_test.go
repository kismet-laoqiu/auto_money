package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"quantlab/internal/platform/notifier"
)

func TestHealthHandlerWritesRuntimeState(t *testing.T) {
	runtime := notifier.NewRuntime(notifier.RuntimeConfig{})
	commandRuntime := notifier.NewCommandRuntime(notifier.CommandRuntimeConfig{
		AllowedChatID: "6959476905",
		Service:       notifier.New(notifier.Config{TelegramBotToken: "demo-token"}),
	})
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	newHealthHandler(runtime, commandRuntime).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var payload struct {
		OK              bool                   `json:"ok"`
		Notifier        notifier.RuntimeHealth `json:"notifier"`
		TelegramCommand notifier.CommandHealth `json:"telegram_command"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !payload.OK || payload.Notifier.ConsumerKey != "notifierd" {
		t.Fatalf("unexpected notifier payload: %+v", payload)
	}
	if !payload.TelegramCommand.Enabled || payload.TelegramCommand.AllowedChatID != "6959476905" {
		t.Fatalf("unexpected telegram command payload: %+v", payload.TelegramCommand)
	}
}
