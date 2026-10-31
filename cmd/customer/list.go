package customer

import (
	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

var listFilters = append([]cmdutil.Field{
	{Param: "name", Flag: "name", Usage: "Filter by customer name (partial match)"},
	{Param: "email", Flag: "email", Usage: "Filter by e-mail address"},
	{Param: "cpfCnpj", Flag: "cpf-cnpj", Usage: "Filter by CPF (11 digits) or CNPJ (14 digits), digits only"},
	{Param: "groupName", Flag: "group-name", Usage: "Filter by customer group name"},
	{Param: "externalReference", Flag: "external-reference", Usage: "Filter by your external reference"},
}, cmdutil.PaginationFields...)

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "list [flags]",
		Short: "List customers, with optional filters and pagination (GET /v3/customers)",
		Long: `List the customers of the authenticated account (GET /v3/customers).

Filters are combined with AND. Results are paginated by the API with
--offset/--limit (max 100 per page); --all follows "hasMore" and returns
every matching customer in a single list (ignores --limit).

Output (stdout), the Asaas list envelope:
  {"object": "list", "hasMore": bool, "totalCount": int, "limit": int,
   "offset": int, "data": [<customer>, ...]}`,
		Example: `  asaas customer list
  asaas customer list --limit 100 --offset 200
  asaas customer list --cpf-cnpj 24971563792
  asaas customer list --name "Marcelo" --all
  ASAAS_API_KEY="$KEY" asaas customer list --sandbox`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmdutil.RunList(cmd, f, basePath, listFilters, all)
		},
	}
	cmdutil.AddFields(cmd, listFilters)
	cmd.Flags().BoolVar(&all, "all", false, "Fetch every page and return all matching customers in one list [type: boolean]")
	return cmd
}
