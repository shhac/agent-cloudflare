package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/cli/shared"
	"github.com/shhac/agent-cloudflare/internal/config"
	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
	"github.com/shhac/agent-cloudflare/internal/output"
)

func registerConfig(root *cobra.Command, globals shared.GlobalsFunc) {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect non-secret CLI configuration",
	}
	configCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show non-secret config",
		RunE: func(cmd *cobra.Command, args []string) error {
			shared.WriteItem(config.Read(), globals().Format)
			return nil
		},
	})
	configCmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Show config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			shared.WriteItem(map[string]any{"path": config.ConfigPath()}, globals().Format)
			return nil
		},
	})
	set := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a non-secret default",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "timeout_ms" {
				output.WriteError(output.Stderr(), agenterrors.Newf(agenterrors.FixableByAgent, "unknown config key %q", args[0]).
					WithHint("Supported keys: timeout_ms"))
				return nil
			}
			var value int
			if _, err := fmtSscanf(args[1], "%d", &value); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByAgent).
					WithHint("Use an integer value, for example: agent-cloudflare config set timeout_ms 10000"))
				return nil
			}
			if err := config.SetDefaultValue(args[0], value); err != nil {
				output.WriteError(output.Stderr(), err)
				return nil
			}
			shared.WriteItem(map[string]any{"status": "set", "key": args[0], "value": value}, globals().Format)
			return nil
		},
	}
	unset := &cobra.Command{
		Use:   "unset <key>",
		Short: "Unset a non-secret default",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.UnsetDefaultValue(args[0]); err != nil {
				output.WriteError(output.Stderr(), err)
				return nil
			}
			shared.WriteItem(map[string]any{"status": "unset", "key": args[0]}, globals().Format)
			return nil
		},
	}
	configCmd.AddCommand(set, unset)
	root.AddCommand(configCmd)
}

var fmtSscanf = fmtSscanfImpl

func fmtSscanfImpl(str, format string, a ...any) (int, error) {
	return fmt.Sscanf(str, format, a...)
}
