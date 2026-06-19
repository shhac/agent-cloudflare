package output

import "testing"

// TestParseFormatAliases pins the lenient parser inherited from
// lib-agent-output: "ndjson" aliases "jsonl", "yml" aliases "yaml", parsing is
// case-insensitive and trims whitespace, and an unknown format still errors.
func TestParseFormatAliases(t *testing.T) {
	cases := []struct {
		in   string
		want Format
		ok   bool
	}{
		{"json", FormatJSON, true},
		{"JSON", FormatJSON, true},
		{"yaml", FormatYAML, true},
		{"yml", FormatYAML, true},
		{"YML", FormatYAML, true},
		{"jsonl", FormatNDJSON, true},
		{"ndjson", FormatNDJSON, true},
		{"  ndjson  ", FormatNDJSON, true},
		{"bogus", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, err := ParseFormat(c.in)
		if c.ok {
			if err != nil {
				t.Errorf("ParseFormat(%q) error = %v, want nil", c.in, err)
				continue
			}
			if got != c.want {
				t.Errorf("ParseFormat(%q) = %q, want %q", c.in, got, c.want)
			}
			continue
		}
		if err == nil {
			t.Errorf("ParseFormat(%q) = %q, want error", c.in, got)
		}
	}
}

// TestResolveFormatOneValueContract pins agent-cloudflare's local one-value
// ResolveFormat: an empty flag yields the default, a valid flag is parsed, and
// an unparseable flag falls back to the default rather than surfacing an error.
func TestResolveFormatOneValueContract(t *testing.T) {
	if got := ResolveFormat("", FormatNDJSON); got != FormatNDJSON {
		t.Errorf("ResolveFormat(empty) = %q, want %q", got, FormatNDJSON)
	}
	if got := ResolveFormat("yaml", FormatJSON); got != FormatYAML {
		t.Errorf("ResolveFormat(yaml) = %q, want %q", got, FormatYAML)
	}
	if got := ResolveFormat("bogus", FormatJSON); got != FormatJSON {
		t.Errorf("ResolveFormat(bogus) = %q, want fallback %q", got, FormatJSON)
	}
}
