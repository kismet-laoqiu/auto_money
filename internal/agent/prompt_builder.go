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

type RiskPacket struct {
	Symbol string
	From   string
	To     string
	Reason string
}

func BuildRiskPrompt(pkt RiskPacket) string {
	return fmt.Sprintf(
		"You are reviewing a trader risk transition. This is advisory only. Do not issue exchange commands. Symbol=%s from=%s to=%s reason=%s.",
		pkt.Symbol,
		pkt.From,
		pkt.To,
		pkt.Reason,
	)
}

type JobRequest struct {
	Kind    string
	Subject string
	Body    string
}

func BuildJobPrompt(job JobRequest) string {
	return fmt.Sprintf(
		"You are operating inside researchd. This is advisory only. Do not issue exchange commands. Job=%s subject=%s. Task=%s",
		job.Kind,
		job.Subject,
		job.Body,
	)
}
