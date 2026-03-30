package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestBuildCandidatePromptCarriesNoExecutionAuthority(t *testing.T) {
	pkt := CandidatePacket{Symbol: "BTCUSDT", SuggestedAction: "probe_long", MaxLeverage: 3}
	prompt := BuildCandidatePrompt(pkt)
	if !strings.Contains(strings.ToLower(prompt), "advisory only") {
		t.Fatalf("prompt must state advisory-only constraint: %q", prompt)
	}
	if strings.Contains(strings.ToLower(prompt), "place order now") {
		t.Fatalf("prompt must not delegate execution authority: %q", prompt)
	}
}

func TestBuildRiskPromptCarriesNoExecutionAuthority(t *testing.T) {
	pkt := RiskPacket{Symbol: "BTCUSDT", From: "armed", To: "degraded", Reason: "position_mismatch"}
	prompt := BuildRiskPrompt(pkt)
	if !strings.Contains(strings.ToLower(prompt), "advisory only") {
		t.Fatalf("prompt must state advisory-only constraint: %q", prompt)
	}
	if strings.Contains(strings.ToLower(prompt), "cancel every order") {
		t.Fatalf("prompt must not delegate execution authority: %q", prompt)
	}
}

func TestServiceReviewCandidateUsesStoredForegroundResponse(t *testing.T) {
	client := &fakeResponsesClient{createResponse: CreateResponse{ID: "resp-review", Status: "completed"}}
	service := NewService(client, Config{Model: "gpt-5.4", Store: true})
	pkt := CandidatePacket{Symbol: "BTCUSDT", SuggestedAction: "probe_long", MaxLeverage: 3}
	resp, err := service.ReviewCandidate(context.Background(), pkt)
	if err != nil {
		t.Fatalf("review candidate: %v", err)
	}
	if resp.ID != "resp-review" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if client.createCalls != 1 || client.backgroundCalls != 0 {
		t.Fatalf("unexpected client usage: %+v", client)
	}
	if !client.lastCreate.Store {
		t.Fatalf("review request must be stored: %+v", client.lastCreate)
	}
	if client.lastCreate.Background {
		t.Fatalf("review request must be foreground: %+v", client.lastCreate)
	}
	if !strings.Contains(strings.ToLower(client.lastCreate.Input), "advisory only") {
		t.Fatalf("review request must carry advisory prompt: %+v", client.lastCreate)
	}
}

func TestServiceExplainRiskUsesStoredForegroundResponse(t *testing.T) {
	client := &fakeResponsesClient{createResponse: CreateResponse{ID: "resp-risk", Status: "completed"}}
	service := NewService(client, Config{Model: "gpt-5.4", Store: true})
	resp, err := service.ExplainRisk(context.Background(), RiskPacket{
		Symbol: "BTCUSDT",
		From:   "armed",
		To:     "degraded",
		Reason: "position_mismatch",
	})
	if err != nil {
		t.Fatalf("explain risk: %v", err)
	}
	if resp.ID != "resp-risk" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if client.createCalls != 1 || client.backgroundCalls != 0 {
		t.Fatalf("unexpected client usage: %+v", client)
	}
	if !client.lastCreate.Store || client.lastCreate.Background {
		t.Fatalf("risk request must be stored foreground: %+v", client.lastCreate)
	}
}

func TestServiceStartBackgroundJobUsesBackgroundEndpoint(t *testing.T) {
	client := &fakeResponsesClient{backgroundResponse: CreateResponse{ID: "resp-job", Status: "in_progress", Background: true}}
	service := NewService(client, Config{Model: "gpt-5.4", Store: true})
	resp, err := service.StartBackgroundJob(context.Background(), JobRequest{
		Kind:    "nightly_report",
		Subject: "BTCUSDT",
		Body:    "Summarize the latest candidate quality drift.",
	})
	if err != nil {
		t.Fatalf("start background job: %v", err)
	}
	if resp.ID != "resp-job" || !resp.Background {
		t.Fatalf("unexpected background response: %+v", resp)
	}
	if client.backgroundCalls != 1 || client.createCalls != 0 {
		t.Fatalf("unexpected client usage: %+v", client)
	}
	if !client.lastBackground.Background {
		t.Fatalf("background request must set background=true: %+v", client.lastBackground)
	}
	if !client.lastBackground.Store {
		t.Fatalf("background request must set store=true: %+v", client.lastBackground)
	}
}

func TestServiceStartBackgroundJobFallsBackToForegroundWhenProviderRejectsBackground(t *testing.T) {
	client := &fakeResponsesClient{
		backgroundErr:  errors.New("openai responses POST https://right.codes/codex/v1/responses: unsupported parameter: background"),
		createResponse: CreateResponse{ID: "resp-sync", Status: "completed"},
	}
	service := NewService(client, Config{Model: "gpt-5.4", Store: true})
	resp, err := service.StartBackgroundJob(context.Background(), JobRequest{
		Kind:    "nightly_report",
		Subject: "mstr-wave-fib",
		Body:    "Prepare the nightly summary.",
	})
	if err != nil {
		t.Fatalf("start background job with fallback: %v", err)
	}
	if resp.ID != "resp-sync" || resp.Status != "completed" {
		t.Fatalf("unexpected fallback response: %+v", resp)
	}
	if client.backgroundCalls != 1 || client.createCalls != 1 {
		t.Fatalf("unexpected client usage: %+v", client)
	}
	if client.lastCreate.Background {
		t.Fatalf("fallback foreground request must clear background: %+v", client.lastCreate)
	}
}

func TestServicePollResponseDelegatesToGet(t *testing.T) {
	client := &fakeResponsesClient{getResponse: CreateResponse{ID: "resp-job", Status: "completed"}}
	service := NewService(client, Config{Model: "gpt-5.4", Store: true})
	resp, err := service.PollResponse(context.Background(), "resp-job")
	if err != nil {
		t.Fatalf("poll response: %v", err)
	}
	if resp.ID != "resp-job" || resp.Status != "completed" {
		t.Fatalf("unexpected poll response: %+v", resp)
	}
	if client.getCalls != 1 {
		t.Fatalf("expected one get call, got %d", client.getCalls)
	}
}

type fakeResponsesClient struct {
	createResponse     CreateResponse
	createErr          error
	backgroundResponse CreateResponse
	backgroundErr      error
	getResponse        CreateResponse
	getErr             error
	lastCreate         CreateRequest
	lastBackground     CreateRequest
	createCalls        int
	backgroundCalls    int
	getCalls           int
}

func (client *fakeResponsesClient) Create(_ context.Context, req CreateRequest) (CreateResponse, error) {
	client.createCalls++
	client.lastCreate = req
	if client.createErr != nil {
		return CreateResponse{}, client.createErr
	}
	return client.createResponse, nil
}

func (client *fakeResponsesClient) CreateBackground(_ context.Context, req CreateRequest) (CreateResponse, error) {
	client.backgroundCalls++
	client.lastBackground = req
	if client.backgroundErr != nil {
		return CreateResponse{}, client.backgroundErr
	}
	return client.backgroundResponse, nil
}

func (client *fakeResponsesClient) Get(_ context.Context, responseID string) (CreateResponse, error) {
	client.getCalls++
	if client.getErr != nil {
		return CreateResponse{}, client.getErr
	}
	if client.getResponse.ID == "" {
		client.getResponse = CreateResponse{ID: responseID}
	}
	return client.getResponse, nil
}
