package config

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/tuliofaria/asaas-cli/pkg/asaasclient"
)

// Credential sources.
const (
	SourceEnv  = "environment_variable"
	SourceFile = "config_file"
)

// Environment sources, i.e. what decided sandbox vs production.
const (
	EnvFromFlag      = "flag"
	EnvFromAccount   = "account"
	EnvFromVariable  = "ASAAS_ENV"
	EnvFromKeyPrefix = "api_key_prefix"
	EnvFromDefault   = "default"
)

// ErrNoCredentials means neither ASAAS_API_KEY nor an active account exist.
var ErrNoCredentials = errors.New("no Asaas credentials configured")

// AccountNotFoundError means a requested account is not in the config file.
type AccountNotFoundError struct {
	Name      string
	Available []string
}

func (e *AccountNotFoundError) Error() string {
	if len(e.Available) == 0 {
		return fmt.Sprintf("account %q not found: no accounts configured", e.Name)
	}
	return fmt.Sprintf("account %q not found (available: %s)", e.Name, strings.Join(e.Available, ", "))
}

// InvalidAccountError means a stored account is unusable.
type InvalidAccountError struct {
	Name   string
	Reason string
}

func (e *InvalidAccountError) Error() string {
	return fmt.Sprintf("account %q is invalid: %s", e.Name, e.Reason)
}

// EnvVarError means an ASAAS_* environment variable has an invalid value.
type EnvVarError struct {
	Name string
	Err  error
}

func (e *EnvVarError) Error() string { return fmt.Sprintf("%s: %v", e.Name, e.Err) }

func (e *EnvVarError) Unwrap() error { return e.Err }

// Overrides are the per-invocation global flags.
type Overrides struct {
	Sandbox    bool
	Production bool
	Account    string
}

// FlagEnvironment returns the environment forced by --sandbox/--production.
func (o Overrides) FlagEnvironment() (asaasclient.Environment, bool) {
	switch {
	case o.Sandbox:
		return asaasclient.Sandbox, true
	case o.Production:
		return asaasclient.Production, true
	}
	return "", false
}

// Credentials is the fully resolved authentication context.
type Credentials struct {
	Source            string
	Account           string
	APIKey            string
	Environment       asaasclient.Environment
	EnvironmentSource string
	BaseURL           string
	ConfigFile        string
}

// Resolve applies the authentication hierarchy:
//
//  1. --account <name>: that account from the config file (explicit flag wins).
//  2. ASAAS_API_KEY: used as-is, the config file is ignored.
//  3. active_account from the config file.
//
// The environment is then taken from --sandbox/--production when given,
// otherwise from the account (file) or ASAAS_ENV / key prefix / production (env).
func Resolve(env Env, store *Store, o Overrides) (*Credentials, error) {
	var creds *Credentials
	var err error

	switch {
	case o.Account != "":
		creds, err = fromFile(store, o.Account)
	case env.APIKey != "":
		creds, err = fromEnv(env)
	default:
		creds, err = fromFile(store, "")
	}
	if err != nil {
		return nil, err
	}

	if e, ok := o.FlagEnvironment(); ok {
		creds.Environment, creds.EnvironmentSource = e, EnvFromFlag
	}
	creds.BaseURL = creds.Environment.BaseURL()
	if env.BaseURL != "" {
		creds.BaseURL = env.BaseURL
	}
	return creds, nil
}

func fromEnv(env Env) (*Credentials, error) {
	c := &Credentials{Source: SourceEnv, APIKey: env.APIKey}
	if env.Environment != "" {
		e, err := asaasclient.ParseEnvironment(env.Environment)
		if err != nil {
			return nil, &EnvVarError{Name: "ASAAS_ENV", Err: err}
		}
		c.Environment, c.EnvironmentSource = e, EnvFromVariable
	} else if e, ok := asaasclient.EnvironmentFromKey(env.APIKey); ok {
		c.Environment, c.EnvironmentSource = e, EnvFromKeyPrefix
	} else {
		c.Environment, c.EnvironmentSource = asaasclient.Production, EnvFromDefault
	}
	return c, nil
}

func fromFile(store *Store, name string) (*Credentials, error) {
	cfg, err := store.Load()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = cfg.ActiveAccount
		if name == "" {
			return nil, ErrNoCredentials
		}
	}
	name, err = NormalizeAccountName(name)
	if err != nil {
		return nil, err
	}
	acc, ok := cfg.Accounts[name]
	if !ok {
		return nil, &AccountNotFoundError{Name: name, Available: cfg.AccountNames()}
	}
	if acc.APIKey == "" {
		return nil, &InvalidAccountError{Name: name, Reason: "api_key is empty"}
	}
	e, err := asaasclient.ParseEnvironment(acc.Environment)
	if err != nil {
		return nil, &InvalidAccountError{Name: name, Reason: err.Error()}
	}
	return &Credentials{
		Source:            SourceFile,
		Account:           name,
		APIKey:            acc.APIKey,
		Environment:       e,
		EnvironmentSource: EnvFromAccount,
		ConfigFile:        store.Path(),
	}, nil
}

// AccountNames returns the configured account names, sorted.
func (c *Config) AccountNames() []string {
	names := make([]string, 0, len(c.Accounts))
	for n := range c.Accounts {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
