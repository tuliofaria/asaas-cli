package customer

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <customer-id>",
		Short: "Delete a customer (DELETE /v3/customers/{id}); reversible with restore",
		Long: `Delete a customer (DELETE /v3/customers/{id}). Asaas performs a soft delete:
the customer is flagged "deleted": true and can be brought back with
'asaas customer restore'. Runs immediately, with no confirmation prompt.

Arguments:
  <customer-id>  Asaas customer ID, e.g. cus_000005219613 [type: string] [required]

Output (stdout): {"deleted": true, "id": "<customer-id>"}`,
		Example: `  asaas customer delete cus_000005219613`,
		Args:    idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.RunRequest(cmd, f, http.MethodDelete, cmdutil.ResourcePath(basePath, args[0]), nil, nil)
		},
	}
}

func newCmdRestore(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "restore <customer-id>",
		Short: "Restore a deleted customer (POST /v3/customers/{id}/restore)",
		Long: `Restore a previously deleted customer (POST /v3/customers/{id}/restore).

Arguments:
  <customer-id>  Asaas customer ID, e.g. cus_000005219613 [type: string] [required]

Output (stdout): the restored customer object.`,
		Example: `  asaas customer restore cus_000005219613`,
		Args:    idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.RunRequest(cmd, f, http.MethodPost, cmdutil.ResourcePath(basePath, args[0], "restore"), nil, nil)
		},
	}
}
