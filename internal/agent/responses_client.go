package agent

import (
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
	var decoded CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return CreateResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return decoded, nil
}
