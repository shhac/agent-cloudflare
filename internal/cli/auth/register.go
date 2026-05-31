package auth

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-cloudflare/internal/cli/shared"
)

func Register(root *cobra.Command, globals shared.GlobalsFunc) {
	profiles := &cobra.Command{
		Use:   "profiles",
		Short: "Manage Cloudflare credential profiles",
	}
	registerProfileCommands(profiles, globals)
	root.AddCommand(profiles)

	authAlias := &cobra.Command{
		Use:    "auth",
		Short:  "Manage Cloudflare credential profiles",
		Hidden: true,
	}
	registerProfileCommands(authAlias, globals)
	root.AddCommand(authAlias)
}

func registerProfileCommands(parent *cobra.Command, globals shared.GlobalsFunc) {
	registerAdd(parent)
	registerUpdate(parent)
	registerCheck(parent, globals)
	registerDiscover(parent, globals)
	registerDefault(parent)
	registerList(parent)
	registerRemove(parent)
}
