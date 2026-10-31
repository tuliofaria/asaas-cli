package auth

import (
	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the accounts stored in the config file (keys masked)",
		Long: `List every account in ~/.config/asaas/config.json, sorted by name.

Output (stdout):
  {"active_account": string, "config_file": string,
   "accounts": [{"name", "environment", "api_key" (masked), "active"}]}`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			store, err := f.Store()
			if err != nil {
				return err
			}
			cfg, err := store.Load()
			if err != nil {
				return err
			}
			accounts := []accountView{}
			for _, n := range cfg.AccountNames() {
				acc := cfg.Accounts[n]
				accounts = append(accounts, maskedView(n, acc.APIKey, acc.Environment, n == cfg.ActiveAccount))
			}
			return f.Print(map[string]any{
				"active_account": cfg.ActiveAccount,
				"config_file":    store.Path(),
				"accounts":       accounts,
			})
		},
	}
}
