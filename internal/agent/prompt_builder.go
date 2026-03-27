package agent

import (
	"fmt"
	"strings"
)

type CandidatePacket struct {
	Symbol          string
	SuggestedAction string
	MaxLeverage     int
	Reasons         []string
}

func BuildCandidatePrompt(pkt CandidatePacket) string {
	base := fmt.Sprintf(
		"You are reviewing a trading candidate. This is advisory only. Do not issue exchange commands. Symbol=%s action=%s max_leverage=%d.",
		pkt.Symbol,
		pkt.SuggestedAction,
		pkt.MaxLeverage,
	)
	if len(pkt.Reasons) == 0 {
		return base
	}
	return base + " Deterministic reasons: " + strings.Join(pkt.Reasons, "; ") + "."
}

type JobRequest struct {
	Kind    string
	Subject string
	Body    string
}

func BuildJobPrompt(job JobRequest) string {
	return fmt.Sprintf(
		"You are operating inside agentd. This is advisory only. Do not issue exchange commands. Job=%s subject=%s. Task=%s",
		job.Kind,
		job.Subject,
		job.Body,
	)
}
