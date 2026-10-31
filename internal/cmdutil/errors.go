package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/config"
	"github.com/tuliofaria/asaas-cli/pkg/asaasclient"
)

// Kind classifies an error in the JSON written to stderr ("type" field).
type Kind string

const (
	KindUsage    Kind = "usage_error"
	KindAuth     Kind = "auth_error"
	KindConfig   Kind = "config_error"
	KindAPI      Kind = "api_error"
	KindNetwork  Kind = "network_error"
	KindResponse Kind = "response_error"
	KindInternal Kind = "internal_error"
)

// Process exit codes, documented in the root command help.
const (
	ExitOK      = 0
	ExitError   = 1 // api_error, response_error, internal_error
	ExitUsage   = 2 // usage_error
	ExitAuth    = 3 // auth_error, config_error
	ExitNetwork = 4 // network_error
)

// Error is a classified CLI error.
type Error struct {
	Kind    Kind
	Message string
	Hint    string
	Err     error
}

func (e *Error) Error() string { return e.Message }

func (e *Error) Unwrap() error { return e.Err }

// UsageErrorf builds a usage_error (invalid flags/arguments).
func UsageErrorf(format string, args ...any) error {
	return &Error{Kind: KindUsage, Message: fmt.Sprintf(format, args...)}
}

// ConfigErrorf builds a config_error.
func ConfigErrorf(format string, args ...any) error {
	return &Error{Kind: KindConfig, Message: fmt.Sprintf(format, args...)}
}

const authHint = "set ASAAS_API_KEY, or run: asaas auth add --name <name> --key <api-key> --sandbox|--production"

// HandleError renders err as {"error": {...}} JSON on w and returns the exit code.
// Errors that are not classified come from Cobra (unknown command/flag,
// missing required flag, wrong number of args) and are reported as usage errors.
func HandleError(w io.Writer, cmd *cobra.Command, err error) int {
	payload := map[string]any{"message": err.Error()}
	code := ExitError

	var (
		cliErr      *Error
		apiErr      *asaasclient.APIError
		netErr      *asaasclient.NetworkError
		respErr     *asaasclient.ResponseError
		fileErr     *config.FileError
		notFoundErr *config.AccountNotFoundError
		invalidErr  *config.InvalidAccountError
		envVarErr   *config.EnvVarError
	)

	switch {
	case errors.As(err, &apiErr):
		payload["type"] = KindAPI
		payload["status_code"] = apiErr.StatusCode
		payload["method"] = apiErr.Method
		payload["path"] = apiErr.Path
		if len(apiErr.Errors) > 0 {
			payload["errors"] = apiErr.Errors
		}
		if apiErr.Body != "" {
			payload["body"] = apiErr.Body
		}
		if apiErr.StatusCode == 401 {
			payload["hint"] = "the API key was rejected; check that it belongs to the selected environment (--sandbox / --production)"
		}
	case errors.As(err, &netErr):
		code = ExitNetwork
		payload["type"] = KindNetwork
		payload["method"] = netErr.Method
		payload["url"] = netErr.URL
		payload["timeout"] = netErr.Timeout()
		if errors.Is(err, context.Canceled) {
			payload["message"] = "request canceled"
		}
	case errors.As(err, &respErr):
		payload["type"] = KindResponse
		payload["status_code"] = respErr.StatusCode
		if respErr.Body != "" {
			payload["body"] = respErr.Body
		}
	case errors.Is(err, config.ErrNoCredentials):
		code = ExitAuth
		payload["type"] = KindAuth
		payload["hint"] = authHint
	case errors.As(err, &notFoundErr):
		code = ExitAuth
		payload["type"] = KindConfig
		payload["available_accounts"] = nonNil(notFoundErr.Available)
	case errors.As(err, &invalidErr), errors.As(err, &fileErr), errors.As(err, &envVarErr):
		code = ExitAuth
		payload["type"] = KindConfig
	case errors.As(err, &cliErr):
		payload["type"] = cliErr.Kind
		if cliErr.Hint != "" {
			payload["hint"] = cliErr.Hint
		}
		code = exitCodeFor(cliErr.Kind)
	default:
		code = ExitUsage
		payload["type"] = KindUsage
	}

	if payload["type"] == KindUsage && cmd != nil {
		payload["command"] = cmd.CommandPath()
		if _, ok := payload["hint"]; !ok {
			payload["hint"] = fmt.Sprintf("run '%s --help' for usage", cmd.CommandPath())
		}
	}

	if perr := PrintJSON(w, map[string]any{"error": payload}); perr != nil {
		fmt.Fprintf(w, "{\"error\":{\"type\":\"internal_error\",\"message\":%q}}\n", err.Error())
	}
	return code
}

func exitCodeFor(k Kind) int {
	switch k {
	case KindUsage:
		return ExitUsage
	case KindAuth, KindConfig:
		return ExitAuth
	case KindNetwork:
		return ExitNetwork
	}
	return ExitError
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
