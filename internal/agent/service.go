package agent

import "context"

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

func (service *Service) StartBackgroundJob(ctx context.Context, job JobRequest) (CreateResponse, error) {
	return service.client.CreateBackground(ctx, CreateRequest{
		Model:      service.cfg.Model,
		Input:      BuildJobPrompt(job),
		Store:      service.cfg.Store,
		Background: true,
	})
}

func (service *Service) PollResponse(ctx context.Context, responseID string) (CreateResponse, error) {
	return service.client.Get(ctx, responseID)
}
