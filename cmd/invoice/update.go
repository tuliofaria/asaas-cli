package invoice

import (
	"bytes"
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
	"github.com/tuliofaria/asaas-cli/pkg/asaasclient"
)

func updateFields() []cmdutil.Field {
	return append([]cmdutil.Field{
		{Param: "serviceDescription", Flag: "service-description", Usage: "Description of the services rendered"},
		{Param: "observations", Flag: "observations", Usage: "Additional notes printed on the invoice"},
		{Param: "value", Flag: "value", Type: cmdutil.Float, Usage: "Total invoice amount in BRL"},
		{Param: "deductions", Flag: "deductions", Type: cmdutil.Float, Usage: "Deductions in BRL (lower the ISS base, not the total)"},
		{Param: "effectiveDate", Flag: "effective-date", Type: cmdutil.Date, Usage: "New issue date"},
		{Param: "externalReference", Flag: "external-reference", Usage: "Your own identifier for this invoice"},
		{Param: "updatePayment", Flag: "update-payment", Type: cmdutil.Bool, Usage: "Reduce the linked payment's value by the withheld taxes"},
	}, taxFields(false)...)
}

func newCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	var data string
	fields := updateFields()
	cmd := &cobra.Command{
		Use:   "update <invoice-id> [flags]",
		Short: "Update a scheduled invoice (PUT /v3/invoices/{id})",
		Long: `Update an invoice that has not been issued yet (PUT /v3/invoices/{id}).
Only flags that are explicitly set are sent. At least one field flag or
--data is required. Origin (payment/customer) and municipal service cannot
be changed; cancel and create a new invoice instead.

Asaas requires the full set of mandatory taxes (retainIss, iss, pis, cofins,
csll, inss, ir) whenever "taxes" is sent. If you change only some of them
(e.g. just --iss), the CLI first reads the invoice and fills the missing
ones with their current values.

Arguments:
  <invoice-id>  Asaas invoice ID, e.g. inv_000000000232 [type: string] [required]

Output (stdout): the updated invoice object.`,
		Example: `  asaas invoice update inv_000000000232 --effective-date 2026-11-10
  asaas invoice update inv_000000000232 --iss 3.5
  asaas invoice update inv_000000000232 --value 450 --observations "Valor revisado"`,
		Args: idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOperationType(cmd); err != nil {
				return err
			}
			set, err := cmdutil.CollectBody(cmd, fields)
			if err != nil {
				return err
			}
			body, err := cmdutil.MergeJSONObject(f.In, "data", data, set)
			if err != nil {
				return err
			}
			if len(body) == 0 {
				return cmdutil.UsageErrorf("nothing to update: set at least one field flag or --data")
			}

			client, err := f.Client()
			if err != nil {
				return err
			}
			path := cmdutil.ResourcePath(basePath, args[0])
			if err := completeTaxes(cmd, client, path, body); err != nil {
				return err
			}
			resp, err := client.Put(cmd.Context(), path, body)
			if err != nil {
				return err
			}
			return f.PrintResponse(resp)
		},
	}
	cmdutil.AddFields(cmd, fields)
	cmd.Flags().StringVar(&data, "data", "", dataUsage)
	return cmd
}

// completeTaxes fills mandatory tax keys missing from body["taxes"] with the
// invoice's current values, since Asaas rejects a partial taxes object.
func completeTaxes(cmd *cobra.Command, client *asaasclient.Client, path string, body map[string]any) error {
	taxes, ok := body["taxes"].(map[string]any)
	if !ok {
		return nil
	}
	complete := true
	for _, k := range taxRequired {
		if _, ok := taxes[k]; !ok {
			complete = false
		}
	}
	if complete {
		return nil
	}

	resp, err := client.Get(cmd.Context(), path, nil)
	if err != nil {
		return err
	}
	var current struct {
		Taxes map[string]any `json:"taxes"`
	}
	dec := json.NewDecoder(bytes.NewReader(resp.Body))
	dec.UseNumber()
	if err := dec.Decode(&current); err != nil {
		return &asaasclient.ResponseError{StatusCode: resp.StatusCode, Message: "reading current invoice taxes: " + err.Error()}
	}
	merged := map[string]any{}
	for _, k := range taxRequired {
		if v, ok := current.Taxes[k]; ok && v != nil {
			merged[k] = v
		}
	}
	cmdutil.DeepMerge(merged, taxes)
	body["taxes"] = merged
	return nil
}
