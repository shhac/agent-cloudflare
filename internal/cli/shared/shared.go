package shared

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"time"

	libcli "github.com/shhac/lib-agent-cli/cli"
	"github.com/shhac/lib-agent-cli/creds"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/config"
	"github.com/shhac/agent-cloudflare/internal/credential"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
	"github.com/shhac/agent-cloudflare/internal/output"
)

type GlobalFlags struct {
	libcli.Globals // Format, TimeoutMS, Debug

	Profile   string
	AccountID string
	ZoneID    string
	Zone      string
	APIToken  string
	BaseURL   string
}

type GlobalsFunc = func() *GlobalFlags

type ResolvedProfile struct {
	Alias     string
	Token     string
	AccountID string
	ZoneID    string
	Zone      string
	Profile   config.Profile
}

func ResolveProfile(flags *GlobalFlags) (*ResolvedProfile, error) {
	if flags == nil {
		flags = &GlobalFlags{}
	}
	apiToken := creds.FirstNonEmpty(flags.APIToken, os.Getenv("CLOUDFLARE_API_TOKEN"), os.Getenv("AGENT_CLOUDFLARE_API_TOKEN"))
	if apiToken != "" {
		return &ResolvedProfile{
			Alias:     "override",
			Token:     apiToken,
			AccountID: flags.AccountID,
			ZoneID:    flags.ZoneID,
			Zone:      flags.Zone,
		}, nil
	}
	cfg := config.Read()
	alias := flags.Profile
	if alias == "" {
		alias = os.Getenv("AGENT_CLOUDFLARE_PROFILE")
	}
	if alias == "" {
		alias = cfg.DefaultProfile
	}
	if alias == "" {
		return nil, agenterrors.New("no Cloudflare profile selected", agenterrors.FixableByHuman).
			WithHint("Run 'agent-cloudflare profiles add <profile> --form' or pass --api-token for a one-shot test")
	}
	profile, ok := cfg.Profiles[alias]
	if !ok {
		return nil, agenterrors.Newf(agenterrors.FixableByHuman, "profile %q is not configured", alias).
			WithHint("Run 'agent-cloudflare profiles list' to see configured profiles")
	}
	token, err := credential.Get(alias)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByHuman).
			WithHint("Re-add the profile with 'agent-cloudflare profiles add " + alias + " --form'")
	}
	accountID := creds.FirstNonEmpty(flags.AccountID, profile.AccountID)
	zoneID := creds.FirstNonEmpty(flags.ZoneID, profile.DefaultZoneID)
	zoneName := creds.FirstNonEmpty(flags.Zone, profile.DefaultZone)
	if flags.Zone != "" && profile.Zones != nil && flags.ZoneID == "" {
		if storedZoneID := profile.Zones[flags.Zone]; storedZoneID != "" {
			zoneID = storedZoneID
		}
	}
	return &ResolvedProfile{
		Alias:     alias,
		Token:     token,
		AccountID: accountID,
		ZoneID:    zoneID,
		Zone:      zoneName,
		Profile:   profile,
	}, nil
}

func WithResolvedClient(flags *GlobalFlags, resolved *ResolvedProfile, fn func(context.Context, *api.Client) error) error {
	ctx, cancel := ContextWithTimeout(context.Background(), flags.TimeoutMS)
	defer cancel()
	client := api.NewClient(api.Options{Token: resolved.Token, BaseURL: flags.BaseURL})
	client.SetDebug(flags.Debug)
	return fn(ctx, client)
}

func WithClient(flags *GlobalFlags, fn func(context.Context, *api.Client, *ResolvedProfile) error) error {
	resolved, err := ResolveProfile(flags)
	if err != nil {
		return err
	}
	return WithResolvedClient(flags, resolved, func(ctx context.Context, client *api.Client) error {
		return fn(ctx, client, resolved)
	})
}

// RequireFlag returns nil when value is set, or a structured fixable_by:agent
// error otherwise. Callers return the error so libcli.Run renders it once and
// exits 1.
func RequireFlag(flag, value, hint string) error {
	if value != "" {
		return nil
	}
	err := agenterrors.Newf(agenterrors.FixableByAgent, "--%s is required", flag)
	if hint != "" {
		err = err.WithHint(hint)
	}
	return err
}

func ContextWithTimeout(parent context.Context, ms int) (context.Context, context.CancelFunc) {
	if ms <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, time.Duration(ms)*time.Millisecond)
}

func WritePaginatedList(items []any, info *api.ResultInfo, format string) {
	if output.ResolveFormat(format, output.FormatNDJSON) == output.FormatNDJSON {
		w := output.NewNDJSONWriter(output.Stdout())
		for _, item := range items {
			_ = w.WriteItem(item)
		}
		if info != nil {
			_ = w.WritePagination(&output.Pagination{
				Page:       info.Page,
				PerPage:    info.PerPage,
				Count:      info.Count,
				TotalCount: info.TotalCount,
				TotalPages: info.TotalPages,
			})
		}
		return
	}
	result := map[string]any{"data": items}
	if info != nil {
		result["pagination"] = info
	}
	output.Print(result, output.ResolveFormat(format, output.FormatJSON), true)
}

func WriteRawPaginatedList(items []json.RawMessage, info *api.ResultInfo, format string) error {
	decoded, err := RawItemsToAny(items)
	if err != nil {
		return err
	}
	WritePaginatedList(decoded, info, format)
	return nil
}

func WriteItem(data any, format string) {
	output.Print(data, output.ResolveFormat(format, output.FormatJSON), true)
}

func WriteRawItem(raw json.RawMessage, format string) {
	output.WriteRawJSON(raw, output.ResolveFormat(format, output.FormatJSON), true)
}

func RawItemsToAny(items []json.RawMessage) ([]any, error) {
	out := make([]any, 0, len(items))
	for _, item := range items {
		var decoded any
		if err := json.Unmarshal(item, &decoded); err != nil {
			return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent).
				WithHint("Cloudflare returned a list item the CLI could not decode; retry with --format json or --debug")
		}
		out = append(out, decoded)
	}
	return out, nil
}

func AddString(values url.Values, key, value string) {
	if value != "" {
		values.Set(key, value)
	}
}
