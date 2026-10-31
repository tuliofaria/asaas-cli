package invoice

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

func newCmdCancel(f *cmdutil.Factory) *cobra.Command {
	var onlyOnAsaas bool
	cmd := &cobra.Command{
		Use:   "cancel <invoice-id> [--only-on-asaas]",
		Short: "Cancel an invoice (POST /v3/invoices/{id}/cancel)",
		Long: `Request cancellation of a service invoice (POST /v3/invoices/{id}/cancel).
Runs with no confirmation prompt.

For an issued (AUTHORIZED) invoice, Asaas asks the city hall to cancel it.
Whether that is automatic depends on the city: check "supportsCancellation"
in 'asaas fiscal-info municipal-options'. The request may stay in
PROCESSING_CANCELLATION before becoming CANCELED or CANCELLATION_DENIED
(webhooks INVOICE_CANCELED / INVOICE_CANCELLATION_DENIED).

--only-on-asaas cancels the record in Asaas only, without contacting the
city hall (e.g. after cancelling it manually on the city's portal).

Arguments:
  <invoice-id>  Asaas invoice ID, e.g. inv_000000000232 [type: string] [required]

Output (stdout): the invoice object.`,
		Example: `  asaas invoice cancel inv_000000000232
  asaas invoice cancel inv_000000000232 --only-on-asaas`,
		Args: idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if cmd.Flags().Changed("only-on-asaas") {
				body["cancelOnlyOnAsaas"] = onlyOnAsaas
			}
			return cmdutil.RunRequest(cmd, f, http.MethodPost, cmdutil.ResourcePath(basePath, args[0], "cancel"), nil, body)
		},
	}
	cmd.Flags().BoolVar(&onlyOnAsaas, "only-on-asaas", false, "Cancel only in Asaas, without asking the city hall to cancel the issued NFS-e [type: boolean]")
	return cmd
}
