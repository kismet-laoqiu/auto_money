package notifier

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
	"slices"
	"strings"
	"testing"
)

func TestServiceSendTelegramUsesBotAPI(t *testing.T) {
	var gotPath string
	var gotChatID string
	var gotText string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		gotPath = request.URL.Path
		var payload struct {
			ChatID string `json:"chat_id"`
			Text   string `json:"text"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode telegram payload: %v", err)
		}
		gotChatID = payload.ChatID
		gotText = payload.Text
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	service := New(Config{
		BaseURL:           server.URL,
		TelegramBotToken:  "demo-token",
		TelegramAlertChat: "6959476905",
		HTTPClient:        server.Client(),
	})
	if err := service.Send(context.Background(), ChannelTelegram, "测试消息"); err != nil {
		t.Fatalf("send telegram: %v", err)
	}
	if gotPath != "/botdemo-token/sendMessage" {
		t.Fatalf("unexpected telegram path: %s", gotPath)
	}
	if gotChatID != "6959476905" || gotText != "测试消息" {
		t.Fatalf("unexpected telegram payload: chat=%s text=%s", gotChatID, gotText)
	}
}

func TestServiceSendDingTalkSignsAndPrefixesKeyword(t *testing.T) {
	var handlerErr string
	var gotContent string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		timestamp := request.URL.Query().Get("timestamp")
		sign := request.URL.Query().Get("sign")
		if timestamp == "" || sign == "" {
			handlerErr = fmt.Sprintf("missing signature query: %s", request.URL.RawQuery)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		mac := hmac.New(sha256.New, []byte("demo-secret"))
		mac.Write([]byte(timestamp + "\n" + "demo-secret"))
		if sign != base64.StdEncoding.EncodeToString(mac.Sum(nil)) {
			handlerErr = fmt.Sprintf("unexpected sign=%s", sign)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			handlerErr = err.Error()
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		var payload struct {
			Text struct {
				Content string `json:"content"`
			} `json:"text"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			handlerErr = err.Error()
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		gotContent = payload.Text.Content
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	service := New(Config{
		DingTalkWebhook: server.URL,
		DingTalkSecret:  "demo-secret",
		DingTalkKeyword: "OTTO",
		HTTPClient:      server.Client(),
	})
	if err := service.Send(context.Background(), ChannelDingTalk, "entry filled"); err != nil {
		t.Fatalf("send dingtalk: %v", err)
	}
	if handlerErr != "" {
		t.Fatal(handlerErr)
	}
	if gotContent != "OTTO\nentry filled" {
		t.Fatalf("unexpected dingtalk content: %q", gotContent)
	}
}

func TestServiceSendTelegramRejectsOKFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":false,"description":"chat not found"}`))
	}))
	defer server.Close()

	service := New(Config{
		BaseURL:           server.URL,
		TelegramBotToken:  "demo-token",
		TelegramAlertChat: "6959476905",
		HTTPClient:        server.Client(),
	})
	if err := service.Send(context.Background(), ChannelTelegram, "测试消息"); err == nil || !strings.Contains(err.Error(), "chat not found") {
		t.Fatalf("expected telegram body error, got %v", err)
	}
}

func TestServiceSendDingTalkRejectsNonZeroErrCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"errcode":310000,"errmsg":"keyword missing"}`))
	}))
	defer server.Close()

	service := New(Config{
		DingTalkWebhook: server.URL,
		HTTPClient:      server.Client(),
	})
	if err := service.Send(context.Background(), ChannelDingTalk, "entry filled"); err == nil || !strings.Contains(err.Error(), "keyword missing") {
		t.Fatalf("expected dingtalk body error, got %v", err)
	}
}

func TestChannelsForKindRiskAndSummary(t *testing.T) {
	if got := ChannelsForKind(KindRiskHalt); !slices.Equal(got, []Channel{ChannelTelegram, ChannelDingTalk}) {
		t.Fatalf("unexpected risk routes: %+v", got)
	}
	if got := ChannelsForKind(KindDailySummary); !slices.Equal(got, []Channel{ChannelTelegram, ChannelDingTalk}) {
		t.Fatalf("unexpected summary routes: %+v", got)
	}
	if got := ChannelsForKind(KindDailySignal); !slices.Equal(got, []Channel{ChannelDingTalk}) {
		t.Fatalf("unexpected daily signal routes: %+v", got)
	}
	if got := ChannelsForKind(KindMarketAlert); !slices.Equal(got, []Channel{ChannelDingTalk}) {
		t.Fatalf("unexpected market alert routes: %+v", got)
	}
}

func TestNotificationRenderUsesChineseHeadingsAndPreservesEnglishTerms(t *testing.T) {
	text := Notification{
		Kind:    KindDailySummary,
		Title:   "每日 summary",
		Summary: "MSTRUSDT fill avgPrice=126.33",
		Details: []string{"strategy=mstr-wave-fib", "status=live"},
	}.Render()
	if !strings.Contains(text, "每日总结") {
		t.Fatalf("expected Chinese heading, got %q", text)
	}
	if !strings.Contains(text, "MSTRUSDT fill avgPrice=126.33") {
		t.Fatalf("expected English trading terms to remain, got %q", text)
	}
	if !strings.Contains(text, "strategy=mstr-wave-fib") {
		t.Fatalf("expected details in rendered text, got %q", text)
	}
}
