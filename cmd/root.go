// Package cmd assembles the asaas command tree.
package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/cmd/api"
	"github.com/tuliofaria/asaas-cli/cmd/auth"
	"github.com/tuliofaria/asaas-cli/cmd/customer"
	"github.com/tuliofaria/asaas-cli/cmd/fiscalinfo"
	"github.com/tuliofaria/asaas-cli/cmd/invoice"
	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

const rootLong = `asaas is a non-interactive command-line client for the Asaas API v3,
designed to be driven by AI agents and automation scripts.

OUTPUT CONTRACT
  - Success: exactly one indented JSON document on stdout (the raw Asaas
    response body, or a CLI-generated object for local commands).
  - Failure: one JSON document on stderr, shaped as
      {"error": {"type": "<kind>", "message": "...", ...}}
    where type is one of: usage_error, auth_error, config_error, api_error,
    network_error, response_error, internal_error. api_error also carries
    status_code and the Asaas "errors" array ([{code, description}]).
  - No colors, no prompts, no pagers.

EXIT CODES
  0 success | 1 api/response/internal error | 2 usage error
  3 auth/config error | 4 network error

AUTHENTICATION (first match wins)
  1. --account <name>      use that account from the config file.
  2. ASAAS_API_KEY         use this key; the config file is ignored.
  3. active_account        from ~/.config/asaas/config.json (see 'asaas auth').

ENVIRONMENT (sandbox vs production)
  --sandbox / --production force the environment for one invocation.
  Otherwise: the account's "environment" (config file), or for ASAAS_API_KEY:
  ASAAS_ENV, then the key prefix ($aact_hmlg_ = sandbox, $aact_prod_ =
  production), then production.
    production: https://api.asaas.com/v3
    sandbox:    https://api-sandbox.asaas.com/v3

ENVIRONMENT VARIABLES
  ASAAS_API_KEY     API key (takes precedence over the config file)
  ASAAS_ENV         "sandbox" or "production", used with ASAAS_API_KEY
  ASAAS_CONFIG_DIR  config directory (default: ~/.config/asaas)
  ASAAS_BASE_URL    override the API base URL (proxies, mocks)

Any endpoint without a dedicated command is reachable through 'asaas api'.`

// NewRootCmd builds the full command tree around a Factory.
func NewRootCmd(f *cmdutil.Factory) *cobra.Command {
	root := &cobra.Command{
		Use:   "asaas <command> <subcommand> [flags]",
		Short: "Asaas API v3 CLI with JSON output, built for AI agents and automation",
		Long:  rootLong,
		Example: `  asaas auth add --name sandbox-dev --key "$KEY" --sandbox
  asaas customer list --limit 5
  ASAAS_API_KEY="$KEY" asaas customer list --sandbox --name "Maria"
  asaas invoice list --status AUTHORIZED
  asaas api /payments --query status=PENDING`,
		SilenceErrors:     true,
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	}
	root.SetOut(f.Out)
	root.SetErr(f.ErrOut)
	root.SetIn(f.In)

	pf := root.PersistentFlags()
	pf.BoolVar(&f.Flags.Sandbox, "sandbox", false,
		"Force the sandbox environment (https://api-sandbox.asaas.com/v3) for this invocation, overriding the account default; with 'auth add', marks the account as sandbox [type: boolean]")
	pf.BoolVar(&f.Flags.Production, "production", false,
		"Force the production environment (https://api.asaas.com/v3) for this invocation, overriding the account default; with 'auth add', marks the account as production [type: boolean]")
	pf.StringVar(&f.Flags.Account, "account", "",
		"Use this config-file account (nickname) for this invocation instead of the active one; takes precedence over ASAAS_API_KEY [type: string]")
	root.MarkFlagsMutuallyExclusive("sandbox", "production")

	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return cmdutil.UsageErrorf("%s", err.Error())
	})

	root.AddCommand(
		auth.NewCmdAuth(f),
		customer.NewCmdCustomer(f),
		invoice.NewCmdInvoice(f),
		fiscalinfo.NewCmdFiscalInfo(f),
		api.NewCmdAPI(f),
		newCmdVersion(f),
	)
	return root
}

// Execute runs the CLI with the process arguments and returns the exit code.
func Execute(version string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return Run(ctx, cmdutil.NewFactory(version), os.Args[1:])
}

// Run executes args against a Factory, rendering any error as JSON on
// f.ErrOut, and returns the exit code.
func Run(ctx context.Context, f *cmdutil.Factory, args []string) int {
	root := NewRootCmd(f)
	root.SetArgs(args)
	cmd, err := root.ExecuteContextC(ctx)
	if err != nil {
		return cmdutil.HandleError(f.ErrOut, cmd, err)
	}
	return cmdutil.ExitOK
}
