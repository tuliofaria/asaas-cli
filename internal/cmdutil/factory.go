// Package cmdutil holds the plumbing shared by every command: dependency
// factory, JSON output, error rendering and declarative flag helpers.
package cmdutil

import (
	"io"
	"net/http"
	"os"

	"github.com/tuliofaria/asaas-cli/internal/config"
	"github.com/tuliofaria/asaas-cli/pkg/asaasclient"
)

// GlobalFlags are the persistent flags defined on the root command.
type GlobalFlags struct {
	Sandbox    bool
	Production bool
	Account    string
}

// Factory provides lazily-built dependencies to commands (gh-style), so a
// command only touches config/credentials/network when it actually runs.
type Factory struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	Version string
	Flags   GlobalFlags

	// LoadEnv reads the ASAAS_* variables; replaceable in tests.
	LoadEnv func() config.Env
	// HTTPClient, when set, is used instead of the default client.
	HTTPClient *http.Client
}

// NewFactory wires the factory to the real process environment.
func NewFactory(version string) *Factory {
	return &Factory{
		In:      os.Stdin,
		Out:     os.Stdout,
		ErrOut:  os.Stderr,
		Version: version,
		LoadEnv: config.LoadEnv,
	}
}

// Env returns the ASAAS_* environment variables.
func (f *Factory) Env() config.Env { return f.LoadEnv() }

// Store returns the config file store.
func (f *Factory) Store() (*config.Store, error) { return config.DefaultStore(f.Env()) }

// Overrides converts the global flags into resolution overrides.
func (f *Factory) Overrides() config.Overrides {
	return config.Overrides{Sandbox: f.Flags.Sandbox, Production: f.Flags.Production, Account: f.Flags.Account}
}

// Credentials resolves which key/environment this invocation uses.
func (f *Factory) Credentials() (*config.Credentials, error) {
	store, err := f.Store()
	if err != nil {
		return nil, err
	}
	return config.Resolve(f.Env(), store, f.Overrides())
}

// Client builds an API client for the resolved credentials.
func (f *Factory) Client() (*asaasclient.Client, error) {
	creds, err := f.Credentials()
	if err != nil {
		return nil, err
	}
	opts := []asaasclient.Option{
		asaasclient.WithUserAgent("asaas-cli/" + f.Version),
		asaasclient.WithBaseURL(creds.BaseURL),
	}
	if f.HTTPClient != nil {
		opts = append(opts, asaasclient.WithHTTPClient(f.HTTPClient))
	}
	c, err := asaasclient.New(creds.APIKey, creds.Environment, opts...)
	if err != nil {
		return nil, &Error{Kind: KindConfig, Message: err.Error(), Err: err}
	}
	return c, nil
}
