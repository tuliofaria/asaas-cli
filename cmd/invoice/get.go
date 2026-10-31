package invoice

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

func newCmdGet(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "get <invoice-id>",
		Short: "Retrieve one invoice by ID (GET /v3/invoices/{id})",
		Long: `Retrieve a single service invoice (GET /v3/invoices/{id}).

Arguments:
  <invoice-id>  Asaas invoice ID, e.g. inv_000000000232 [type: string] [required]

Output (stdout): the invoice object. Check "status" (SCHEDULED, AUTHORIZED,
PROCESSING_CANCELLATION, CANCELED, CANCELLATION_DENIED, ERROR), and
"statusDescription" for the city hall's message when it is ERROR. Once
AUTHORIZED, "pdfUrl" and "xmlUrl" point to the issued document.`,
		Example: `  asaas invoice get inv_000000000232`,
		Args:    idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.RunRequest(cmd, f, http.MethodGet, cmdutil.ResourcePath(basePath, args[0]), nil, nil)
		},
	}
}

func newCmdAuthorize(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "authorize <invoice-id>",
		Short: "Issue a scheduled invoice now (POST /v3/invoices/{id}/authorize)",
		Long: `Send a scheduled invoice to the city hall / Portal Nacional immediately,
instead of waiting for its effectiveDate (POST /v3/invoices/{id}/authorize).

This issues a legal fiscal document and runs with no confirmation prompt.
Authorization is asynchronous: the response may still show a processing
status; poll 'asaas invoice get' until AUTHORIZED or ERROR, or listen to the
INVOICE_AUTHORIZED / INVOICE_ERROR webhooks.

Arguments:
  <invoice-id>  Asaas invoice ID, e.g. inv_000000000232 [type: string] [required]

Output (stdout): the invoice object.`,
		Example: `  asaas invoice authorize inv_000000000232`,
		Args:    idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.RunRequest(cmd, f, http.MethodPost, cmdutil.ResourcePath(basePath, args[0], "authorize"), nil, nil)
		},
	}
}
