package cli

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

const trafficQuery = `
query AgentCloudflareTraffic($zoneTag: string, $start: Time, $end: Time, $limit: uint64) {
  viewer {
    zones(filter: { zoneTag: $zoneTag }) {
      series: httpRequestsAdaptiveGroups(
        limit: $limit
        orderBy: [datetimeHour_ASC]
        filter: { datetime_geq: $start, datetime_lt: $end }
      ) {
        dimensions {
          datetimeHour
          edgeResponseStatus
          cacheStatus
          clientRequestHTTPHost
        }
        count
        sum {
          edgeResponseBytes
          visits
        }
      }
    }
  }
}`

func registerAnalytics(root *cobra.Command, globals shared.GlobalsFunc) {
	var since string
	var limit int

	analytics := &cobra.Command{
		Use:   "analytics",
		Short: "Read Cloudflare analytics through GraphQL",
	}
	traffic := &cobra.Command{
		Use:   "traffic [zone-name-or-id]",
		Short: "Summarize HTTP traffic for a zone",
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
				start, end, err := sinceWindow(since, time.Now())
				if err != nil {
					return err
				}
				raw, err := client.GraphQL(ctx, trafficQuery, map[string]any{
					"zoneTag": zoneID,
					"start":   start.Format(time.RFC3339),
					"end":     end.Format(time.RFC3339),
					"limit":   limit,
				})
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, flags.Format)
				return nil
			})
		},
	}
	traffic.Flags().StringVar(&since, "since", "1h", "Lookback duration, such as 15m, 1h, or 24h")
	traffic.Flags().IntVar(&limit, "limit", 100, "Maximum GraphQL groups to return")
	analytics.AddCommand(traffic)
	root.AddCommand(analytics)
}
