package cli

import (
	"strings"

	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	"github.com/shhac/agent-cloudflare/internal/output"
)

type evidenceRecord struct {
	Type     string `json:"type"`
	Object   string `json:"object,omitempty"`
	ID       string `json:"id,omitempty"`
	Severity string `json:"severity,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Data     any    `json:"data,omitempty"`
}

func writeEvidence(records []evidenceRecord, format string) {
	if output.ResolveFormat(format, output.FormatNDJSON) == output.FormatNDJSON {
		w := output.NewNDJSONWriter(output.Stdout())
		for _, record := range records {
			_ = w.WriteItem(record)
		}
		return
	}
	shared.WriteItem(map[string]any{"records": records}, format)
}

func errorFinding(object, severity, summary string, err error) evidenceRecord {
	return evidenceRecord{
		Type:     "finding",
		Object:   object,
		Severity: severity,
		Summary:  summary,
		Data: map[string]any{
			"error": strings.TrimSpace(err.Error()),
		},
	}
}
