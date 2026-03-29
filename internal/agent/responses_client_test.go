package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPResponsesClientCreateEncodesStringInputAsMessageArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		input, ok := payload["input"].([]any)
		if !ok {
			t.Fatalf("expected input array, got %#v", payload["input"])
		}
		if len(input) != 1 {
			t.Fatalf("expected one input message, got %d", len(input))
		}

		message, ok := input[0].(map[string]any)
		if !ok {
			t.Fatalf("expected message object, got %#v", input[0])
		}
		if message["role"] != "user" {
			t.Fatalf("unexpected role: %#v", message["role"])
		}
		if message["content"] != "review this candidate" {
			t.Fatalf("unexpected content: %#v", message["content"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp-1","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`))
	}))
	defer server.Close()

	client := NewHTTPResponsesClient(HTTPClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		HTTPClient: server.Client(),
	})
	resp, err := client.Create(context.Background(), CreateRequest{
		Model: "gpt-5.4",
		Input: "review this candidate",
		Store: true,
	})
	if err != nil {
		t.Fatalf("create response: %v", err)
	}
	if resp.ID != "resp-1" || resp.Status != "completed" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.OutputText() != "ok" {
		t.Fatalf("unexpected output text: %q", resp.OutputText())
	}
}

func TestHTTPResponsesClientCreateDecodesCompletedEventStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: response.created\n")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp-stream\",\"status\":\"in_progress\"}}\n\n")
		_, _ = io.WriteString(w, "event: response.completed\n")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp-stream\",\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"stream ok\"}]}]}}\n\n")
	}))
	defer server.Close()

	client := NewHTTPResponsesClient(HTTPClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		HTTPClient: server.Client(),
	})
	resp, err := client.Create(context.Background(), CreateRequest{
		Model: "gpt-5.4",
		Input: "decode event stream",
		Store: true,
	})
	if err != nil {
		t.Fatalf("create response: %v", err)
	}
	if resp.ID != "resp-stream" || resp.Status != "completed" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.OutputText() != "stream ok" {
		t.Fatalf("unexpected output text: %q", resp.OutputText())
	}
}
