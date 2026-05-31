package cli

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/api"
	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func registerRawAPI(root *cobra.Command, globals shared.GlobalsFunc) {
	var queryPairs []string
	var printRequest bool

	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Read-only raw Cloudflare API escape hatch",
	}
	get := &cobra.Command{
		Use:   "get <path>",
		Short: "GET a Cloudflare API path with active credentials",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := parseQueryPairs(queryPairs)
			if err != nil {
				return err
			}
			path := buildRawPath(args[0], params)
			flags := globals()
			return shared.WithClient(flags, func(ctx context.Context, client *api.Client, resolved *shared.ResolvedProfile) error {
				if printRequest {
					shared.WriteItem(client.PreviewRequest(http.MethodGet, path, nil), flags.Format)
					return nil
				}
				raw, _, err := client.RawRequest(ctx, http.MethodGet, path, nil)
				if err != nil {
					return err
				}
				shared.WriteRawItem(raw, flags.Format)
				return nil
			})
		},
	}
	get.Flags().StringArrayVar(&queryPairs, "query", nil, "Query parameter as k=v; repeatable")
	get.Flags().BoolVar(&printRequest, "print-request", false, "Print redacted request preview without sending")
	apiCmd.AddCommand(get)
	root.AddCommand(apiCmd)
}

func parseQueryPairs(pairs []string) (url.Values, error) {
	values := url.Values{}
	for _, pair := range pairs {
		key, value, ok := splitQueryPair(pair)
		if !ok {
			return nil, agenterrors.Newf(agenterrors.FixableByAgent, "invalid --query value %q", pair).
				WithHint("Use --query key=value, for example --query name=example.com")
		}
		values.Add(key, value)
	}
	return values, nil
}

func splitQueryPair(pair string) (string, string, bool) {
	for i, r := range pair {
		if r == '=' {
			return pair[:i], pair[i+1:], i > 0
		}
	}
	return "", "", false
}

func buildRawPath(path string, params url.Values) string {
	if encoded := params.Encode(); encoded != "" {
		if strings.Contains(path, "?") {
			return path + "&" + encoded
		}
		return path + "?" + encoded
	}
	return path
}
