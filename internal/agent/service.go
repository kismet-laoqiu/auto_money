package agent

import (
	"context"
	"strings"
)

type Config struct {
	Model string
	Store bool
}

type Service struct {
	client ResponsesClient
	cfg    Config
}

func NewService(client ResponsesClient, cfg Config) *Service {
	if cfg.Model == "" {
		cfg.Model = defaultModel
	}
	if !cfg.Store {
		cfg.Store = true
	}
	return &Service{client: client, cfg: cfg}
}

func (service *Service) ReviewCandidate(ctx context.Context, pkt CandidatePacket) (CreateResponse, error) {
	return service.client.Create(ctx, CreateRequest{
		Model: service.cfg.Model,
		Input: BuildCandidatePrompt(pkt),
		Store: service.cfg.Store,
	})
}

func (service *Service) ExplainRisk(ctx context.Context, pkt RiskPacket) (CreateResponse, error) {
	return service.client.Create(ctx, CreateRequest{
		Model: service.cfg.Model,
		Input: BuildRiskPrompt(pkt),
		Store: service.cfg.Store,
	})
}

func (service *Service) StartBackgroundJob(ctx context.Context, job JobRequest) (CreateResponse, error) {
	request := CreateRequest{
		Model:      service.cfg.Model,
		Input:      BuildJobPrompt(job),
		Store:      service.cfg.Store,
		Background: true,
	}
	response, err := service.client.CreateBackground(ctx, request)
	if err == nil || !shouldFallbackBackground(err) {
		return response, err
	}
	request.Background = false
	return service.client.Create(ctx, request)
}

func shouldFallbackBackground(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unsupported parameter") && strings.Contains(message, "background")
}

func (service *Service) PollResponse(ctx context.Context, responseID string) (CreateResponse, error) {
	return service.client.Get(ctx, responseID)
}
