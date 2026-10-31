// Package api implements "asaas api": raw authenticated requests to any
// Asaas v3 endpoint, in the spirit of "gh api".
package api

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

type apiOptions struct {
	method    string
	query     []string
	fields    []string
	rawFields []string
	data      string
	all       bool
}

var allowedMethods = map[string]bool{
	http.MethodGet: true, http.MethodPost: true, http.MethodPut: true,
	http.MethodDelete: true, http.MethodPatch: true,
}

// NewCmdAPI builds the "api" command.
func NewCmdAPI(f *cmdutil.Factory) *cobra.Command {
	opts := &apiOptions{}
	cmd := &cobra.Command{
		Use:   "api <path> [flags]",
		Short: "Send an authenticated request to any Asaas API v3 endpoint",
		Long: `Send an authenticated request to any Asaas API v3 endpoint and print the
JSON response. Use it for resources that have no dedicated command yet
(payments, subscriptions, pix, transfers, webhooks, ...).

Arguments:
  <path>  Endpoint path relative to /v3, e.g. /payments or payments/pay_123.
          A leading "/v3" is accepted and stripped. Absolute URLs are rejected
          so the API key is only ever sent to the Asaas host. [type: string] [required]

Request body (POST/PUT/PATCH):
  --data       a JSON document (inline, @file.json, or - for stdin)
  -F/--field   key=value with JSON typing: true, false, null and valid JSON
               numbers become literals ("0123" stays a string)
  -f/--raw-field key=value, always sent as a string (use for CPF/CNPJ, CEP...)
  Fields are merged on top of --data, which must then be a JSON object.

Output (stdout): the raw Asaas response body, or
{"success": true, "status_code": N} when the API returns no body.`,
		Example: `  asaas api /payments --query status=PENDING --query limit=20
  asaas api /payments -X POST -f customer=cus_000005219613 -f billingType=PIX \
    -F value=129.90 -f dueDate=2026-12-31
  asaas api /subscriptions -X POST --data @subscription.json
  asaas api /finance/balance
  asaas api /payments --all --query customer=cus_000005219613`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmdutil.UsageErrorf("expected exactly one <path> argument (e.g. /payments), got %d", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAPI(cmd, f, opts, args[0])
		},
	}

	fl := cmd.Flags()
	fl.StringVarP(&opts.method, "method", "X", http.MethodGet, "HTTP method: GET, POST, PUT, DELETE or PATCH [type: string]")
	fl.StringArrayVarP(&opts.query, "query", "q", nil, "Query parameter as key=value; repeatable [type: string, key=value]")
	fl.StringArrayVarP(&opts.fields, "field", "F", nil, "Body field as key=value with JSON typing (true/false/null/numbers); repeatable [type: string, key=value]")
	fl.StringArrayVarP(&opts.rawFields, "raw-field", "f", nil, "Body field as key=value, always a JSON string; repeatable [type: string, key=value]")
	fl.StringVarP(&opts.data, "data", "d", "", "Request body as JSON: inline, @file.json, or - for stdin [type: JSON]")
	fl.BoolVar(&opts.all, "all", false, "GET only: follow pagination (hasMore) and return every item of a list endpoint in one list [type: boolean]")
	return cmd
}

func runAPI(cmd *cobra.Command, f *cmdutil.Factory, opts *apiOptions, path string) error {
	method := strings.ToUpper(opts.method)
	if !allowedMethods[method] {
		return cmdutil.UsageErrorf("--method: unsupported HTTP method %q", opts.method)
	}

	query := url.Values{}
	for _, kv := range opts.query {
		k, v, err := cmdutil.ParseKeyValue("query", kv)
		if err != nil {
			return err
		}
		query.Add(k, v)
	}

	fields := map[string]any{}
	for _, kv := range opts.fields {
		k, v, err := cmdutil.ParseKeyValue("field", kv)
		if err != nil {
			return err
		}
		fields[k] = cmdutil.TypedValue(v)
	}
	for _, kv := range opts.rawFields {
		k, v, err := cmdutil.ParseKeyValue("raw-field", kv)
		if err != nil {
			return err
		}
		fields[k] = v
	}

	hasBody := opts.data != "" || len(fields) > 0
	if hasBody && (method == http.MethodGet || method == http.MethodDelete) {
		return cmdutil.UsageErrorf("%s requests take no body; use --query for parameters", method)
	}
	if opts.all && method != http.MethodGet {
		return cmdutil.UsageErrorf("--all can only be used with GET")
	}

	var body any
	switch {
	case len(fields) > 0:
		merged, err := cmdutil.MergeJSONObject(f.In, "data", opts.data, fields)
		if err != nil {
			return err
		}
		body = merged
	case opts.data != "":
		raw, err := cmdutil.ReadJSONArg(f.In, "data", opts.data)
		if err != nil {
			return err
		}
		body = raw
	}

	if opts.all {
		client, err := f.Client()
		if err != nil {
			return err
		}
		page, err := client.ListAll(cmd.Context(), path, query)
		if err != nil {
			return err
		}
		return f.Print(page)
	}
	return cmdutil.RunRequest(cmd, f, method, path, query, body)
}
