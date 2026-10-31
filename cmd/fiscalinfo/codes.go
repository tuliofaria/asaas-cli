package fiscalinfo

import (
	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

// newListCmd builds a paginated lookup command over a /fiscalInfo list endpoint.
func newListCmd(f *cmdutil.Factory, use, short, long, example, path string, filters []cmdutil.Field) *cobra.Command {
	var all bool
	filters = append(filters, cmdutil.PaginationFields...)
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmdutil.RunList(cmd, f, path, filters, all)
		},
	}
	cmdutil.AddFields(cmd, filters)
	cmd.Flags().BoolVar(&all, "all", false, "Fetch every page and return all matching items in one list [type: boolean]")
	return cmd
}

func newCmdServices(f *cmdutil.Factory) *cobra.Command {
	return newListCmd(f, "services [--description <text>]",
		"Search the city hall's service list for a municipalServiceId (GET /v3/fiscalInfo/services)",
		`Search the municipal services registered for the account's city
(GET /v3/fiscalInfo/services). Use the returned "id" as
'asaas invoice create --municipal-service-id'.

Not available on the Portal Nacional: there, use --municipal-service-code
(see 'fiscal-info federal-service-codes').

Output (stdout): the Asaas list envelope; each item has "id",
"description" and "issTax".`,
		`  asaas fiscal-info services --description "desenvolvimento"
  asaas fiscal-info services --all`,
		basePath+"/services",
		[]cmdutil.Field{
			{Param: "description", Flag: "description", Usage: "Filter by service name (partial match)"},
		})
}

func newCmdNBSCodes(f *cmdutil.Factory) *cobra.Command {
	return newListCmd(f, "nbs-codes [--search <text>]",
		"Search NBS codes (Nomenclatura Brasileira de Serviços) (GET /v3/fiscalInfo/nbsCodes)",
		`Search NBS codes (GET /v3/fiscalInfo/nbsCodes), used in
'fiscal-info save --nbs-code' and 'invoice create --nbs-code'.

Output (stdout): the Asaas list envelope of NBS codes and descriptions.`,
		`  asaas fiscal-info nbs-codes --search "software"
  asaas fiscal-info nbs-codes --search 1.1502`,
		basePath+"/nbsCodes",
		[]cmdutil.Field{
			{Param: "codeDescription", Flag: "search", Usage: "Filter by NBS code or description (partial match)"},
		})
}

func newCmdFederalServiceCodes(f *cmdutil.Factory) *cobra.Command {
	return newListCmd(f, "federal-service-codes [--code <code>] [--description <text>]",
		"Search national service tax codes (código de tributação nacional) (GET /v3/fiscalInfo/federalServiceCodes)",
		`Search the national service tax codes (código de tributação nacional,
federalServiceTaxCode) used by the Portal Nacional da NFS-e
(GET /v3/fiscalInfo/federalServiceCodes).

Output (stdout): the Asaas list envelope of codes and descriptions.`,
		`  asaas fiscal-info federal-service-codes --description "programação"
  asaas fiscal-info federal-service-codes --code 01.07`,
		basePath+"/federalServiceCodes",
		[]cmdutil.Field{
			{Param: "code", Flag: "code", Usage: "Filter by tax code (e.g. 01.07.01)"},
			{Param: "description", Flag: "description", Usage: "Filter by description (partial match)"},
		})
}
