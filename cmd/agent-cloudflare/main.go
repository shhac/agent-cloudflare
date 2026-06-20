package main

import (
	libcli "github.com/shhac/lib-agent-cli/cli"

	"github.com/shhac/agent-cloudflare/internal/cli"
)

var version = "dev"

func main() {
	libcli.Run(cli.NewRootCmd(version))
}
