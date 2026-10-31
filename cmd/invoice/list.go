package invoice

import (
	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

var listFilters = append([]cmdutil.Field{
	{Param: "effectiveDate[ge]", Flag: "effective-date-from", Type: cmdutil.Date, Usage: "Only invoices with effectiveDate on or after this date"},
	{Param: "effectiveDate[le]", Flag: "effective-date-to", Type: cmdutil.Date, Usage: "Only invoices with effectiveDate on or before this date"},
	{Param: "status", Flag: "status", Enum: statuses, Usage: "Filter by status"},
	{Param: "customer", Flag: "customer", Usage: "Filter by customer ID (cus_...)"},
	{Param: "payment", Flag: "payment", Usage: "Filter by payment ID (pay_...)"},
	{Param: "installment", Flag: "installment", Usage: "Filter by installment ID"},
	{Param: "externalReference", Flag: "external-reference", Usage: "Filter by your external reference"},
}, cmdutil.PaginationFields...)

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "list [flags]",
		Short: "List invoices, with optional filters and pagination (GET /v3/invoices)",
		Long: `List the service invoices of the authenticated account (GET /v3/invoices).

Filters are combined with AND. Results are paginated by the API with
--offset/--limit (max 100 per page); --all follows "hasMore" and returns
every matching invoice in a single list (ignores --limit).

Output (stdout), the Asaas list envelope:
  {"object": "list", "hasMore": bool, "totalCount": int, "limit": int,
   "offset": int, "data": [<invoice>, ...]}`,
		Example: `  asaas invoice list --status AUTHORIZED --limit 20
  asaas invoice list --effective-date-from 2026-10-01 --effective-date-to 2026-10-31 --all
  asaas invoice list --customer cus_000005219613
  asaas invoice list --payment pay_080225913252`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmdutil.RunList(cmd, f, basePath, listFilters, all)
		},
	}
	cmdutil.AddFields(cmd, listFilters)
	cmd.Flags().BoolVar(&all, "all", false, "Fetch every page and return all matching invoices in one list [type: boolean]")
	return cmd
}
