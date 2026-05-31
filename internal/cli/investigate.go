package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func registerInvestigate(root *cobra.Command, globals shared.GlobalsFunc) {
	var since string

	investigate := &cobra.Command{
		Use:   "investigate",
		Short: "Gather Cloudflare evidence for common operational questions",
	}
	investigate.AddCommand(&cobra.Command{
		Use:   "usage",
		Short: "Show investigation command examples",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), `agent-cloudflare investigate usage

Available investigations:
  agent-cloudflare investigate zone-health <zone-name-or-id>
  agent-cloudflare investigate traffic-spike <zone-name-or-id> --since 1h
  agent-cloudflare investigate dns-change <zone-name-or-id>
  agent-cloudflare investigate ssl-breakage <zone-name-or-id>
  agent-cloudflare investigate waf-block <zone-name-or-id>
  agent-cloudflare investigate worker-error --account-id <account_id>
  agent-cloudflare investigate cache-miss <zone-name-or-id>

Output:
  Investigation records default to NDJSON evidence rows.
  Finding rows use severity: info, warning, critical.
`)
			return nil
		},
	})
	zoneHealth := &cobra.Command{
		Use:   "zone-health [zone-name-or-id]",
		Short: "Gather zone, DNS, SSL/TLS, cache, rulesets, and Waiting Room evidence",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				zoneRef := resolved.ZoneID
				if len(args) > 0 {
					zoneRef = args[0]
				}
				zoneID, err := resolveZoneID(ctx, client, resolved, zoneRef)
				if err != nil {
					return err
				}
				records := investigateZoneHealth(ctx, client, zoneID)
				writeEvidence(records, flags.Format)
				return nil
			})
		},
	}
	investigate.AddCommand(zoneHealth)

	trafficSpike := &cobra.Command{
		Use:   "traffic-spike [zone-name-or-id]",
		Short: "Gather analytics and audit evidence for a traffic spike",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZoneInvestigation(globals(), args, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) ([]evidenceRecord, error) {
				start, end, err := sinceWindow(since, time.Now())
				if err != nil {
					return nil, err
				}
				return investigateTrafficSpike(ctx, client, resolved, zoneID, start, end)
			})
		},
	}
	trafficSpike.Flags().StringVar(&since, "since", "1h", "Lookback duration, such as 15m, 1h, or 24h")

	investigate.AddCommand(trafficSpike)
	investigate.AddCommand(zoneInvestigationCommand("dns-change", "Gather DNS and audit evidence for recent DNS changes", globals, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) ([]evidenceRecord, error) {
		return investigateDNSChange(ctx, client, resolved, zoneID), nil
	}))
	investigate.AddCommand(zoneInvestigationCommand("ssl-breakage", "Gather SSL/TLS evidence for certificate or HTTPS issues", globals, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) ([]evidenceRecord, error) {
		settings, findings := collectSettingsSoft(ctx, client, zoneID, sslSettingIDs)
		records := []evidenceRecord{{Type: "entity", Object: "ssl_settings", ID: zoneID, Data: settings}}
		records = append(records, findings...)
		records = append(records, sslFindings(settings)...)
		return records, nil
	}))
	investigate.AddCommand(zoneInvestigationCommand("waf-block", "Gather rulesets and traffic evidence for suspected WAF blocks", globals, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) ([]evidenceRecord, error) {
		return investigateWAFBlock(ctx, client, zoneID), nil
	}))
	investigate.AddCommand(zoneInvestigationCommand("cache-miss", "Gather cache settings and traffic evidence for cache misses", globals, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, zoneID string) ([]evidenceRecord, error) {
		return investigateCacheMiss(ctx, client, zoneID), nil
	}))
	investigate.AddCommand(accountInvestigationCommand("worker-error", "Gather Workers evidence for account-level Worker errors", globals, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile, accountID string) ([]evidenceRecord, error) {
		return investigateWorkerError(ctx, client, accountID), nil
	}))
	root.AddCommand(investigate)
}

type zoneInvestigationFunc func(context.Context, *api.Client, *shared.ResolvedProfile, string) ([]evidenceRecord, error)
type accountInvestigationFunc func(context.Context, *api.Client, *shared.ResolvedProfile, string) ([]evidenceRecord, error)

func zoneInvestigationCommand(use, short string, globals shared.GlobalsFunc, fn zoneInvestigationFunc) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [zone-name-or-id]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZoneInvestigation(globals(), args, fn)
		},
	}
}

func accountInvestigationCommand(use, short string, globals shared.GlobalsFunc, fn accountInvestigationFunc) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				accountID, err := requireAccountID(resolved)
				if err != nil {
					return err
				}
				records, err := fn(ctx, client, resolved, accountID)
				if err != nil {
					return err
				}
				writeEvidence(records, flags.Format)
				return nil
			})
		},
	}
}

func runZoneInvestigation(flags *shared.GlobalFlags, args []string, fn zoneInvestigationFunc) error {
	return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
		zoneRef := resolved.ZoneID
		if len(args) > 0 {
			zoneRef = args[0]
		}
		zoneID, err := resolveZoneID(ctx, client, resolved, zoneRef)
		if err != nil {
			return err
		}
		records, err := fn(ctx, client, resolved, zoneID)
		if err != nil {
			return err
		}
		writeEvidence(records, flags.Format)
		return nil
	})
}
