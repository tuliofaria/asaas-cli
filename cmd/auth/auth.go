// Package auth implements "asaas auth": local multi-account management.
package auth

import (
	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
	"github.com/tuliofaria/asaas-cli/pkg/asaasclient"
)

// NewCmdAuth builds the "auth" command group.
func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	cmd := cmdutil.NewGroup("auth <subcommand>",
		"Manage Asaas accounts (API keys) stored in the local config file",
		`Manage the accounts stored in ~/.config/asaas/config.json
(or $ASAAS_CONFIG_DIR/config.json). The file maps account nicknames to an
API key and an environment, plus the currently active account:

  {
    "active_account": "sandbox-dev",
    "accounts": {
      "sandbox-dev":    {"api_key": "$aact_hmlg_...", "environment": "sandbox"},
      "producao-firma": {"api_key": "$aact_prod_...", "environment": "production"}
    }
  }

The file is written with 0600 permissions. When ASAAS_API_KEY is set, API
commands use it and ignore this file; 'auth add/switch/list/remove' still
edit the file.`)

	cmd.AddCommand(
		newCmdAdd(f),
		newCmdSwitch(f),
		newCmdStatus(f),
		newCmdList(f),
		newCmdRemove(f),
	)
	return cmd
}

// accountView is the JSON shape used to describe an account.
type accountView struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
	APIKey      string `json:"api_key"`
	Active      bool   `json:"active"`
}

func maskedView(name, key, env string, active bool) accountView {
	return accountView{Name: name, Environment: env, APIKey: asaasclient.MaskKey(key), Active: active}
}
