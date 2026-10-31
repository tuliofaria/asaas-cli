package auth

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
	"github.com/tuliofaria/asaas-cli/internal/config"
	"github.com/tuliofaria/asaas-cli/pkg/asaasclient"
)

func newCmdStatus(f *cmdutil.Factory) *cobra.Command {
	var verify bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show which account, key and environment API commands will use",
		Long: `Resolve the authentication context exactly as API commands do (honoring
ASAAS_API_KEY, --account, --sandbox and --production) and print it. The API
key is masked: only its environment prefix and last 4 characters are shown.

Output (stdout):
  {
    "source": "environment_variable" | "config_file",
    "account": string (config_file only),
    "environment": "sandbox" | "production",
    "environment_source": "flag" | "account" | "ASAAS_ENV" | "api_key_prefix" | "default",
    "base_url": string,
    "api_key": string (masked),
    "config_file": string,
    "verified": bool (only with --verify)
  }

With --verify, a real request (GET /customers?limit=1) checks that the key is
accepted; a rejected key returns api_error (exit 1) on stderr. Without
credentials, fails with auth_error (exit 3).`,
		Example: `  asaas auth status
  asaas auth status --verify
  ASAAS_API_KEY="$KEY" asaas auth status --sandbox`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			creds, err := f.Credentials()
			if err != nil {
				return err
			}
			out := map[string]any{
				"source":             creds.Source,
				"environment":        creds.Environment,
				"environment_source": creds.EnvironmentSource,
				"base_url":           creds.BaseURL,
				"api_key":            asaasclient.MaskKey(creds.APIKey),
			}
			if creds.Source == config.SourceFile {
				out["account"] = creds.Account
				out["config_file"] = creds.ConfigFile
			} else if store, err := f.Store(); err == nil {
				out["config_file"] = store.Path()
				out["note"] = "ASAAS_API_KEY is set; the config file is ignored"
			}

			if verify {
				client, err := f.Client()
				if err != nil {
					return err
				}
				if _, err := client.Get(cmd.Context(), "/customers", url.Values{"limit": {"1"}}); err != nil {
					return err
				}
				out["verified"] = true
			}
			return f.Print(out)
		},
	}
	cmd.Flags().BoolVar(&verify, "verify", false, "Also send a real API request to confirm the key is accepted [type: boolean]")
	return cmd
}
