package auth

import (
	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
	"github.com/tuliofaria/asaas-cli/internal/config"
)

func newCmdRemove(f *cmdutil.Factory) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "remove --name <name>",
		Short: "Remove an account from the config file",
		Long: `Delete an account from ~/.config/asaas/config.json. Removing the active
account leaves "active_account" empty; pick another with 'asaas auth switch'.

Output (stdout):
  {"removed": string, "active_account": string, "config_file": string}`,
		Example: `  asaas auth remove --name sandbox-dev`,
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
			if _, ok := cfg.Accounts[n]; !ok {
				return &config.AccountNotFoundError{Name: n, Available: cfg.AccountNames()}
			}
			delete(cfg.Accounts, n)
			if cfg.ActiveAccount == n {
				cfg.ActiveAccount = ""
			}
			if err := store.Save(cfg); err != nil {
				return err
			}
			return f.Print(map[string]any{
				"removed":        n,
				"active_account": cfg.ActiveAccount,
				"config_file":    store.Path(),
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Nickname of the account to remove [type: string]")
	cmdutil.MarkRequired(cmd, "name")
	return cmd
}
