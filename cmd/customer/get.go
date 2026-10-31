package customer

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

func newCmdGet(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "get <customer-id>",
		Short: "Retrieve one customer by ID (GET /v3/customers/{id})",
		Long: `Retrieve a single customer (GET /v3/customers/{id}).

Arguments:
  <customer-id>  Asaas customer ID, e.g. cus_000005219613 [type: string] [required]

Output (stdout): the customer object. Deleted customers are still returned,
with "deleted": true. Unknown IDs return api_error with status_code 404.`,
		Example: `  asaas customer get cus_000005219613`,
		Args:    idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.RunRequest(cmd, f, http.MethodGet, cmdutil.ResourcePath(basePath, args[0]), nil, nil)
		},
	}
}
