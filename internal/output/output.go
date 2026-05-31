package output

import (
	"encoding/json"
	"io"
	"os"

	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatJSON   Format = "json"
	FormatYAML   Format = "yaml"
	FormatNDJSON Format = "jsonl"
)

const (
	MetaKeyPagination = "@pagination"
	MetaKeyRequest    = "@request"
)

var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

func SetWriters(out, err io.Writer) func() {
	prevOut := stdout
	prevErr := stderr
	stdout = out
	stderr = err
	return func() {
		stdout = prevOut
		stderr = prevErr
	}
}

func Stdout() io.Writer { return stdout }
func Stderr() io.Writer { return stderr }

func ParseFormat(s string) (Format, error) {
	switch s {
	case "json":
		return FormatJSON, nil
	case "yaml":
		return FormatYAML, nil
	case "jsonl", "ndjson":
		return FormatNDJSON, nil
	default:
		return "", agenterrors.Newf(agenterrors.FixableByAgent, "unknown format %q, expected: json, yaml, jsonl", s)
	}
}

func ResolveFormat(flagFormat string, defaultFormat Format) Format {
	if flagFormat == "" {
		return defaultFormat
	}
	f, err := ParseFormat(flagFormat)
	if err != nil {
		return defaultFormat
	}
	return f
}

func Print(data any, format Format, prune bool) {
	switch format {
	case FormatYAML:
		printYAML(data, prune)
	default:
		printJSON(data, prune)
	}
}

func PrintJSON(data any, prune bool) {
	printJSON(data, prune)
}

func WriteRawJSON(raw json.RawMessage, format Format, indent bool) {
	if format == FormatNDJSON {
		_, _ = Stdout().Write(raw)
		_, _ = Stdout().Write([]byte("\n"))
		return
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		_, _ = Stdout().Write(raw)
		_, _ = Stdout().Write([]byte("\n"))
		return
	}
	Print(data, format, true)
}

func printJSON(data any, prune bool) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	var decoded any
	if err := json.Unmarshal(b, &decoded); err != nil {
		return
	}
	if prune {
		decoded = pruneNulls(decoded)
	}
	enc := json.NewEncoder(Stdout())
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(decoded)
}

func printYAML(data any, prune bool) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	var decoded any
	if err := json.Unmarshal(b, &decoded); err != nil {
		return
	}
	if prune {
		decoded = pruneNulls(decoded)
	}
	enc := yaml.NewEncoder(Stdout())
	enc.SetIndent(2)
	_ = enc.Encode(decoded)
}

func WriteError(w io.Writer, err error) {
	var aerr *agenterrors.APIError
	if !agenterrors.As(err, &aerr) {
		aerr = agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	payload := map[string]any{
		"error":      aerr.Message,
		"fixable_by": string(aerr.FixableBy),
	}
	if aerr.Hint != "" {
		payload["hint"] = aerr.Hint
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}

type NDJSONWriter struct {
	enc *json.Encoder
}

func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &NDJSONWriter{enc: enc}
}

func (n *NDJSONWriter) WriteItem(item any) error {
	return n.enc.Encode(item)
}

func (n *NDJSONWriter) WriteMetaLine(key string, value any) error {
	return n.enc.Encode(map[string]any{key: value})
}

type Pagination struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Count      int `json:"count,omitempty"`
	TotalCount int `json:"total_count,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

func (n *NDJSONWriter) WritePagination(p *Pagination) error {
	return n.WriteMetaLine(MetaKeyPagination, p)
}

func pruneNulls(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v := range val {
			if v == nil {
				continue
			}
			out[k] = pruneNulls(v)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, v := range val {
			out[i] = pruneNulls(v)
		}
		return out
	default:
		return v
	}
}
