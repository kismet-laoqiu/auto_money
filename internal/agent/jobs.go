package agent

import "context"

type JobStatus struct {
	ResponseID string
	Status     string
	Text       string
}

func (service *Service) StartNightlyReport(ctx context.Context, subject, task string) (CreateResponse, error) {
	return service.StartBackgroundJob(ctx, JobRequest{
		Kind:    "nightly_report",
		Subject: subject,
		Body:    task,
	})
}

func (service *Service) PollJob(ctx context.Context, responseID string) (JobStatus, error) {
	resp, err := service.PollResponse(ctx, responseID)
	if err != nil {
		return JobStatus{}, err
	}
	return JobStatus{ResponseID: resp.ID, Status: resp.Status, Text: resp.OutputText()}, nil
}
