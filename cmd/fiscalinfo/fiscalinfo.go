// Package fiscalinfo implements "asaas fiscal-info": the account's fiscal
// configuration for issuing service invoices (/v3/fiscalInfo).
package fiscalinfo

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

const basePath = "/fiscalInfo"

// NewCmdFiscalInfo builds the "fiscal-info" command group.
func NewCmdFiscalInfo(f *cmdutil.Factory) *cobra.Command {
	cmd := cmdutil.NewGroup("fiscal-info <subcommand>",
		"Configure invoice (NFS-e) issuing: fiscal data, city requirements, service and NBS codes",
		`Manage the fiscal configuration that service invoices ('asaas invoice')
depend on (/v3/fiscalInfo).

Typical setup:
  1. 'fiscal-info municipal-options'  what the account's city hall requires
     (authenticationType: CERTIFICATE | TOKEN | USER_AND_PASSWORD, special
     tax regimes, service list item, whether cancellation is automatic).
  2. 'fiscal-info save'               send the company's fiscal data and the
     city hall credentials matching authenticationType.
  3. 'fiscal-info services'           find the municipalServiceId for invoices
     (or 'federal-service-codes' / 'nbs-codes' for national codes).
  4. 'fiscal-info national-portal'    switch issuing to the Portal Nacional
     da NFS-e, when applicable.`)

	cmd.AddCommand(
		newCmdGet(f),
		newCmdSave(f),
		newCmdMunicipalOptions(f),
		newCmdServices(f),
		newCmdNBSCodes(f),
		newCmdFederalServiceCodes(f),
		newCmdNationalPortal(f),
	)
	return cmd
}

func newCmdGet(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show the account's fiscal information (GET /v3/fiscalInfo)",
		Long: `Show the fiscal information used to issue invoices (GET /v3/fiscalInfo).

Output (stdout): {"object", "email", "municipalInscription", "simplesNacional",
"culturalProjectsPromoter", "cnae", "specialTaxRegime", "serviceListItem",
"nbsCode", "rpsSerie", "rpsNumber", "loteNumber", "username", "passwordSent",
"accessTokenSent", "certificateSent", "nationalPortalTaxCalculationRegime",
"serviceProvisionCityDefaultType"}. Secrets are never returned, only the
*Sent booleans. Returns api_error 404 when nothing was configured yet.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmdutil.RunRequest(cmd, f, http.MethodGet, basePath+"/", nil, nil)
		},
	}
}

func newCmdMunicipalOptions(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "municipal-options",
		Short: "Show what the account's city hall requires to issue invoices (GET /v3/fiscalInfo/municipalOptions)",
		Long: `Show the city hall's requirements for the account's CNPJ city
(GET /v3/fiscalInfo/municipalOptions). Read this before 'fiscal-info save'.

Output (stdout):
  {"authenticationType": "CERTIFICATE" | "TOKEN" | "USER_AND_PASSWORD",
   "supportsCancellation": bool, "usesSpecialTaxRegimes": bool,
   "usesServiceListItem": bool,
   "specialTaxRegimesList": [{"label", "value"}],
   "nationalPortalTaxCalculationRegimeList": [{"label", "value"}],
   "...Help": string hints about each field's format}`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmdutil.RunRequest(cmd, f, http.MethodGet, basePath+"/municipalOptions", nil, nil)
		},
	}
}

func newCmdNationalPortal(f *cmdutil.Factory) *cobra.Command {
	var enabled bool
	cmd := &cobra.Command{
		Use:   "national-portal --enabled=<true|false>",
		Short: "Enable or disable issuing through the Portal Nacional da NFS-e (POST /v3/fiscalInfo/nationalPortal)",
		Long: `Enable or disable issuing invoices through the Portal Nacional da NFS-e
instead of the city hall's own system (POST /v3/fiscalInfo/nationalPortal).

On the Portal Nacional there is no municipal service list: create invoices
with --municipal-service-code instead of --municipal-service-id.

Output (stdout): the Asaas response.`,
		Example: `  asaas fiscal-info national-portal --enabled=true
  asaas fiscal-info national-portal --enabled=false`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmdutil.RunRequest(cmd, f, http.MethodPost, basePath+"/nationalPortal", nil, map[string]any{"enabled": enabled})
		},
	}
	cmd.Flags().BoolVar(&enabled, "enabled", false, "true to issue through the Portal Nacional, false to use the city hall system [type: boolean]")
	cmdutil.MarkRequired(cmd, "enabled")
	return cmd
}
