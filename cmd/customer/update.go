package customer

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

func newCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	var data string
	fields := bodyFields(false)
	cmd := &cobra.Command{
		Use:   "update <customer-id> [flags]",
		Short: "Update fields of an existing customer (PUT /v3/customers/{id})",
		Long: `Update a customer (PUT /v3/customers/{id}). Only flags that are explicitly
set are sent, so unspecified fields keep their current values. At least one
field flag or --data is required.

Arguments:
  <customer-id>  Asaas customer ID, e.g. cus_000005219613 [type: string] [required]

Output (stdout): the updated customer object.`,
		Example: `  asaas customer update cus_000005219613 --email novo@email.com
  asaas customer update cus_000005219613 --notification-disabled=true
  asaas customer update cus_000005219613 --data '{"observations":"VIP"}'`,
		Args: idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			return cmdutil.RunRequest(cmd, f, http.MethodPut, cmdutil.ResourcePath(basePath, args[0]), nil, body)
		},
	}
	cmdutil.AddFields(cmd, fields)
	cmd.Flags().StringVar(&data, "data", "", dataUsage)
	return cmd
}
