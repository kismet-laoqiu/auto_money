package bitget

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.bitget.com"

type Client struct {
	baseURL    string
	httpClient *http.Client
	creds      PrivateCredentials
	signer     *Signer
}

func NewClient(baseURL string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func NewPrivateClient(baseURL string, creds PrivateCredentials) *Client {
	client := NewClient(baseURL)
	client.creds = creds
	client.signer = NewSigner(creds.Secret)
	return client
}

func (client *Client) doPublic(ctx context.Context, method, path string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("%s %s: status=%d body=%s", request.Method, request.URL.String(), response.StatusCode, string(body))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (client *Client) doPrivate(ctx context.Context, method, path string, payload any) ([]byte, error) {
	if strings.TrimSpace(client.creds.Key) == "" || strings.TrimSpace(client.creds.Secret) == "" || strings.TrimSpace(client.creds.Passphrase) == "" {
		return nil, fmt.Errorf("bitget private credentials are incomplete")
	}
	var bodyBytes []byte
	if payload != nil {
		var err error
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}
	timestamp := strconvFormatInt(time.Now().UTC().UnixMilli())
	request.Header.Set("ACCESS-KEY", client.creds.Key)
	request.Header.Set("ACCESS-PASSPHRASE", client.creds.Passphrase)
	request.Header.Set("ACCESS-TIMESTAMP", timestamp)
	request.Header.Set("ACCESS-SIGN", client.signer.Sign(timestamp, method, path, string(bodyBytes)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("locale", "en-US")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s: status=%d body=%s", request.Method, request.URL.String(), response.StatusCode, string(body))
	}
	return body, nil
}

func strconvFormatInt(value int64) string {
	return fmt.Sprintf("%d", value)
}
