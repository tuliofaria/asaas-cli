package auth

import (
	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
	"github.com/tuliofaria/asaas-cli/internal/config"
)

func newCmdSwitch(f *cmdutil.Factory) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "switch --name <name>",
		Short: "Change the active account in the config file",
		Long: `Set "active_account" in ~/.config/asaas/config.json to an existing account.
Subsequent API commands use it unless ASAAS_API_KEY or --account is given.

Output (stdout):
  {"account": {"name", "environment", "api_key" (masked), "active": true},
   "previous_active_account": string, "config_file": string}

Fails with config_error (exit 3) and "available_accounts" when the name is unknown.`,
		Example: `  asaas auth switch --name producao-firma`,
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			n, err := config.NormalizeAccountName(name)
			if err != nil {
				return cmdutil.UsageErrorf("--name: %v", err)
			}
			store, err := f.Store()
			if err != nil {
				return err
			}
			cfg, err := store.Load()
			if err != nil {
				return err
			}
			acc, ok := cfg.Accounts[n]
			if !ok {
				return &config.AccountNotFoundError{Name: n, Available: cfg.AccountNames()}
			}
			previous := cfg.ActiveAccount
			cfg.ActiveAccount = n
			if err := store.Save(cfg); err != nil {
				return err
			}
			out := map[string]any{
				"account":                 maskedView(n, acc.APIKey, acc.Environment, true),
				"previous_active_account": previous,
				"config_file":             store.Path(),
			}
			if f.Env().APIKey != "" {
				out["warning"] = "ASAAS_API_KEY is set and takes precedence over the active account for API commands"
			}
			return f.Print(out)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Nickname of an existing account to activate [type: string]")
	cmdutil.MarkRequired(cmd, "name")
	return cmd
}
