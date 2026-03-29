package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-5.4"
)

type ResponsesClient interface {
	Create(ctx context.Context, req CreateRequest) (CreateResponse, error)
	CreateBackground(ctx context.Context, req CreateRequest) (CreateResponse, error)
	Get(ctx context.Context, responseID string) (CreateResponse, error)
}

type CreateRequest struct {
	Model      string            `json:"model"`
	Input      string            `json:"input"`
	Store      bool              `json:"store"`
	Background bool              `json:"background,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type requestInputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type createRequestWire struct {
	Model      string                `json:"model"`
	Input      []requestInputMessage `json:"input"`
	Store      bool                  `json:"store"`
	Background bool                  `json:"background,omitempty"`
	Metadata   map[string]string     `json:"metadata,omitempty"`
}

func (req CreateRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(createRequestWire{
		Model: req.Model,
		Input: []requestInputMessage{{
			Role:    "user",
			Content: req.Input,
		}},
		Store:      req.Store,
		Background: req.Background,
		Metadata:   req.Metadata,
	})
}

type CreateResponse struct {
	ID         string       `json:"id"`
	Status     string       `json:"status"`
	Background bool         `json:"background,omitempty"`
	Output     []OutputItem `json:"output,omitempty"`
}

type OutputItem struct {
	Type    string        `json:"type"`
	Content []ContentPart `json:"content,omitempty"`
}

type ContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func (response CreateResponse) OutputText() string {
	var parts []string
	for _, item := range response.Output {
		for _, content := range item.Content {
			if content.Text != "" {
				parts = append(parts, content.Text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

type HTTPClientConfig struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

type HTTPResponsesClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewHTTPResponsesClient(cfg HTTPClientConfig) *HTTPResponsesClient {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &HTTPResponsesClient{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		httpClient: httpClient,
	}
}

func (client *HTTPResponsesClient) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	req.Background = false
	return client.doJSON(ctx, http.MethodPost, client.baseURL+"/responses", req)
}

func (client *HTTPResponsesClient) CreateBackground(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	req.Background = true
	return client.doJSON(ctx, http.MethodPost, client.baseURL+"/responses", req)
}

func (client *HTTPResponsesClient) Get(ctx context.Context, responseID string) (CreateResponse, error) {
	return client.doJSON(ctx, http.MethodGet, client.baseURL+"/responses/"+url.PathEscape(responseID), nil)
}

func (client *HTTPResponsesClient) doJSON(ctx context.Context, method, endpoint string, payload any) (CreateResponse, error) {
	if client.apiKey == "" {
		return CreateResponse{}, fmt.Errorf("openai api key is empty")
	}
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return CreateResponse{}, fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return CreateResponse{}, fmt.Errorf("openai responses %s %s: %s", method, endpoint, strings.TrimSpace(string(data)))
	}
	decoded, err := decodeCreateResponse(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		return CreateResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return decoded, nil
}

type responseStreamEnvelope struct {
	Type     string         `json:"type"`
	Response CreateResponse `json:"response"`
}

func decodeCreateResponse(body io.Reader, contentType string) (CreateResponse, error) {
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") {
		return decodeCreateResponseStream(body)
	}
	var decoded CreateResponse
	if err := json.NewDecoder(body).Decode(&decoded); err != nil {
		return CreateResponse{}, err
	}
	return decoded, nil
}

func decodeCreateResponseStream(body io.Reader) (CreateResponse, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var eventType string
	var dataLines []string
	var lastResponse CreateResponse

	flush := func() (CreateResponse, bool, error) {
		if len(dataLines) == 0 {
			eventType = ""
			return CreateResponse{}, false, nil
		}

		payload := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = nil
		currentType := eventType
		eventType = ""

		if payload == "" || payload == "[DONE]" {
			return CreateResponse{}, false, nil
		}

		var envelope responseStreamEnvelope
		if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
			return CreateResponse{}, false, fmt.Errorf("stream event %s: %w", currentType, err)
		}
		if envelope.Response.ID != "" {
			lastResponse = envelope.Response
		}
		switch {
		case currentType == "response.completed" || envelope.Type == "response.completed":
			return envelope.Response, true, nil
		case currentType == "response.failed" || envelope.Type == "response.failed":
			return CreateResponse{}, false, fmt.Errorf("response failed with status %s", envelope.Response.Status)
		default:
			return CreateResponse{}, false, nil
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			response, done, err := flush()
			if err != nil {
				return CreateResponse{}, err
			}
			if done {
				return response, nil
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "event:"):
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return CreateResponse{}, err
	}

	response, done, err := flush()
	if err != nil {
		return CreateResponse{}, err
	}
	if done {
		return response, nil
	}
	if lastResponse.ID != "" && strings.EqualFold(lastResponse.Status, "completed") {
		return lastResponse, nil
	}
	return CreateResponse{}, fmt.Errorf("no completed response event in stream")
}
