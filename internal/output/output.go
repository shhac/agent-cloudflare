// Package output re-exports the shared output contract from lib-agent-output,
// keeping the internal/output import path while the wire mechanism (format
// parsing, JSON/YAML encoding, error rendering) lives in one place. What stays
// local is agent-cloudflare policy: the writer indirection used by tests, the
// Cloudflare-shaped pagination trailer, NDJSON raw passthrough, and the
// one-value ResolveFormat contract its callers depend on. (Migration shim.)
package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	out "github.com/shhac/lib-agent-output"
	"gopkg.in/yaml.v3"
)

// Format and its values come from the shared contract; ParseFormat is therefore
// the family's lenient parser (accepts "ndjson"/"yml", case-insensitive).
type Format = out.Format

const (
	FormatJSON   = out.FormatJSON
	FormatYAML   = out.FormatYAML
	FormatNDJSON = out.FormatNDJSON
)

const (
	MetaKeyPagination = "@pagination"
	MetaKeyRequest    = "@request"
)

var (
	ParseFormat = out.ParseFormat
	WriteError  = out.WriteError
)

// init registers agent-cloudflare's YAML encoder with lib-agent-output, so YAML
// support (and its yaml.v3 dependency) stays in this CLI while the core library
// remains dependency-free.
func init() {
	out.RegisterEncoder(out.FormatYAML, func(v any) ([]byte, error) {
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(v); err != nil {
			return nil, err
		}
		_ = enc.Close()
		return buf.Bytes(), nil
	})
}

var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

func SetWriters(o, e io.Writer) func() {
	prevOut := stdout
	prevErr := stderr
	stdout = o
	stderr = e
	return func() {
		stdout = prevOut
		stderr = prevErr
	}
}

func Stdout() io.Writer { return stdout }
func Stderr() io.Writer { return stderr }

// ResolveFormat keeps agent-cloudflare's one-value contract: an unparseable
// flag falls back to the default rather than surfacing an error (callers treat
// format selection as best-effort).
func ResolveFormat(flagFormat string, defaultFormat Format) Format {
	f, err := out.ResolveFormat(flagFormat, defaultFormat)
	if err != nil {
		return defaultFormat
	}
	return f
}

// Print prunes nulls (opt-in) then encodes data in the given format via the
// shared encoder, to the indirected stdout writer.
func Print(data any, format Format, prune bool) {
	_ = out.Print(Stdout(), data, format, pruner(prune))
}

func PrintJSON(data any, prune bool) {
	Print(data, FormatJSON, prune)
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

// Pagination is Cloudflare-shaped (page/per_page/count totals), so it stays
// local rather than using out.Pagination.
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

// pruner maps the legacy bool prune flag onto a shared Pruner: prune nulls when
// set, no pruning otherwise (preserving exact encoding).
func pruner(prune bool) out.Pruner {
	if !prune {
		return nil
	}
	return out.PruneNils
}
