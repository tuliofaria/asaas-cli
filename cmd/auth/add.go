package auth

import (
	"bufio"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
	"github.com/tuliofaria/asaas-cli/internal/config"
	"github.com/tuliofaria/asaas-cli/pkg/asaasclient"
)

type addOptions struct {
	name     string
	key      string
	force    bool
	activate bool
}

func newCmdAdd(f *cmdutil.Factory) *cobra.Command {
	opts := &addOptions{}
	cmd := &cobra.Command{
		Use:   "add --name <name> --key <api-key> [--sandbox | --production]",
		Short: "Add an Asaas account (nickname + API key + environment) to the config file",
		Long: `Add an account to ~/.config/asaas/config.json and make it the active account.

The environment is taken from the global --sandbox or --production flag. When
neither is given it is inferred from the key prefix ($aact_hmlg_ = sandbox,
$aact_prod_ = production); if the key has no known prefix the command fails
and asks for an explicit flag. A flag that contradicts the key prefix is an
error.

Use --key - to read the key from stdin, keeping it out of shell history and
the process list. The key is not validated against the API; run
'asaas auth status --verify' afterwards to check it.

Output (stdout):
  {"account": {"name", "environment", "api_key" (masked), "active"},
   "replaced": bool, "previous_active_account": string, "config_file": string}`,
		Example: `  asaas auth add --name sandbox-dev --key "$ASAAS_SANDBOX_KEY" --sandbox
  printf '%s' "$KEY" | asaas auth add --name producao-firma --key - --production
  asaas auth add --name sandbox-dev --key "$NEW_KEY" --sandbox --force`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAdd(f, opts)
		},
	}

	fl := cmd.Flags()
	fl.StringVar(&opts.name, "name", "", "Account nickname, 1-64 chars of [a-z0-9_-], case-insensitive (e.g. sandbox-dev) [type: string]")
	fl.StringVar(&opts.key, "key", "", "Asaas API key, or '-' to read it from stdin [type: string]")
	fl.BoolVar(&opts.force, "force", false, "Overwrite the account if the nickname already exists [type: boolean]")
	fl.BoolVar(&opts.activate, "activate", true, "Make this the active account; use --activate=false to keep the current one [type: boolean]")
	cmdutil.MarkRequired(cmd, "name", "key")
	return cmd
}

func runAdd(f *cmdutil.Factory, opts *addOptions) error {
	name, err := config.NormalizeAccountName(opts.name)
	if err != nil {
		return cmdutil.UsageErrorf("--name: %v", err)
	}

	key := opts.key
	if key == "-" {
		line, err := bufio.NewReader(f.In).ReadString('\n')
		if err != nil && line == "" {
			return cmdutil.UsageErrorf("--key -: could not read the API key from stdin: %v", err)
		}
		key = line
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return cmdutil.UsageErrorf("--key: API key is empty")
	}

	env, err := accountEnvironment(f.Overrides(), key)
	if err != nil {
		return err
	}

	store, err := f.Store()
	if err != nil {
		return err
	}
	cfg, err := store.Load()
	if err != nil {
		return err
	}

	_, exists := cfg.Accounts[name]
	if exists && !opts.force {
		return &cmdutil.Error{
			Kind:    cmdutil.KindConfig,
			Message: "account " + name + " already exists",
			Hint:    "pass --force to overwrite it",
		}
	}

	previous := cfg.ActiveAccount
	cfg.Accounts[name] = config.Account{APIKey: key, Environment: string(env)}
	_, previousValid := cfg.Accounts[previous]
	if opts.activate || previous == "" || !previousValid {
		cfg.ActiveAccount = name
	}
	if err := store.Save(cfg); err != nil {
		return err
	}

	out := map[string]any{
		"account":                 maskedView(name, key, string(env), cfg.ActiveAccount == name),
		"replaced":                exists,
		"previous_active_account": previous,
		"config_file":             store.Path(),
	}
	if f.Env().APIKey != "" {
		out["warning"] = "ASAAS_API_KEY is set and takes precedence over the config file for API commands"
	}
	return f.Print(out)
}

// accountEnvironment picks the environment for a new account from the global
// flags or the key prefix, rejecting contradictions.
func accountEnvironment(o config.Overrides, key string) (asaasclient.Environment, error) {
	flagEnv, hasFlag := o.FlagEnvironment()
	keyEnv, hasPrefix := asaasclient.EnvironmentFromKey(key)
	switch {
	case hasFlag && hasPrefix && flagEnv != keyEnv:
		return "", cmdutil.UsageErrorf("the API key prefix indicates a %s key, but --%s was given", keyEnv, flagEnv)
	case hasFlag:
		return flagEnv, nil
	case hasPrefix:
		return keyEnv, nil
	}
	return "", cmdutil.UsageErrorf("cannot infer the environment from the API key; pass --sandbox or --production")
}
