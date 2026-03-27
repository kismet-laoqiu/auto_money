package agent

type MCPTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReadOnly    bool   `json:"read_only"`
}

func ReadOnlyToolSurface() []MCPTool {
	return []MCPTool{
		{
			Name:        "candidate.explain",
			Description: "Explain deterministic candidate packets without execution authority.",
			ReadOnly:    true,
		},
		{
			Name:        "risk.explain",
			Description: "Explain degraded or halted states without mutating exchange state.",
			ReadOnly:    true,
		},
		{
			Name:        "report.nightly",
			Description: "Prepare nightly research summaries as advisory-only background jobs.",
			ReadOnly:    true,
		},
	}
}
