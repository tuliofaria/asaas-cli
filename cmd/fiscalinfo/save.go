package fiscalinfo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
	"github.com/tuliofaria/asaas-cli/pkg/asaasclient"
)

var saveFields = []cmdutil.Field{
	{Param: "email", Flag: "email", BodyRequired: true, Usage: "E-mail that receives invoice notifications and alerts from Asaas"},
	{Param: "simplesNacional", Flag: "simples-nacional", Type: cmdutil.Bool, BodyRequired: true, Usage: "Whether the company is in the Simples Nacional regime; pass --simples-nacional=false explicitly when not"},
	{Param: "municipalInscription", Flag: "municipal-inscription", Usage: "Company's municipal registration (inscrição municipal)"},
	{Param: "culturalProjectsPromoter", Flag: "cultural-projects-promoter", Type: cmdutil.Bool, Usage: "Whether the company is a cultural incentive promoter (incentivador cultural)"},
	{Param: "cnae", Flag: "cnae", Usage: "CNAE code"},
	{Param: "specialTaxRegime", Flag: "special-tax-regime", Usage: "Special tax regime ID; a value from specialTaxRegimesList in 'fiscal-info municipal-options'"},
	{Param: "serviceListItem", Flag: "service-list-item", Usage: "Service list item per LC 116/2003, when usesServiceListItem is true"},
	{Param: "nbsCode", Flag: "nbs-code", Usage: "Default NBS code; see 'fiscal-info nbs-codes'"},
	{Param: "rpsSerie", Flag: "rps-serie", Usage: "RPS series used by the company (some cities require 10001-49999)"},
	{Param: "rpsNumber", Flag: "rps-number", Type: cmdutil.Int, Usage: "RPS number of the last invoice issued (Asaas continues from the next one)"},
	{Param: "loteNumber", Flag: "lote-number", Type: cmdutil.Int, Usage: "Batch (lote) number of the last invoice issued"},
	{Param: "username", Flag: "username", Usage: "City hall portal username (authenticationType USER_AND_PASSWORD)"},
	{Param: "password", Flag: "password", Usage: "City hall portal password, or '-' to read it from stdin (USER_AND_PASSWORD)"},
	{Param: "accessToken", Flag: "access-token", Usage: "City hall portal token, or '-' to read it from stdin (authenticationType TOKEN)"},
	{Param: "certificatePassword", Flag: "certificate-password", Usage: "Digital certificate password, or '-' to read it from stdin (authenticationType CERTIFICATE)"},
	{Param: "nationalPortalTaxCalculationRegime", Flag: "national-portal-tax-calculation-regime", Usage: "Simples Nacional tax calculation regime on the Portal Nacional; a value from nationalPortalTaxCalculationRegimeList"},
	{Param: "serviceProvisionCityDefaultType", Flag: "service-provision-city-default-type", Enum: []string{"MY_ACCOUNT_CITY", "CUSTOMER_CITY", "CUSTOMER_CITY_WHEN_ISS_IS_RETAINED"}, Usage: "Default city where the service is provided; kept as-is when omitted"},
}

// secretFlags accept "-" to read the value from stdin.
var secretFlags = []string{"password", "access-token", "certificate-password"}

func newCmdSave(f *cmdutil.Factory) *cobra.Command {
	var certFile string
	cmd := &cobra.Command{
		Use:   "save --email <email> --simples-nacional=<bool> [flags]",
		Short: "Create or update the account's fiscal information (POST /v3/fiscalInfo)",
		Long: `Create or update the fiscal information used to issue invoices
(POST /v3/fiscalInfo, sent as multipart/form-data).

Run 'asaas fiscal-info municipal-options' first and send only the city hall
credentials matching its authenticationType:
  CERTIFICATE        --certificate-file <file.pfx> --certificate-password <pwd>
  TOKEN              --access-token <token>
  USER_AND_PASSWORD  --username <user> --password <pwd>
Provide --special-tax-regime when usesSpecialTaxRegimes is true and
--service-list-item when usesServiceListItem is true.

--email and --simples-nacional are required on every call. Only flags that
are explicitly set are sent.

Secrets: --password, --access-token and --certificate-password accept '-'
to read the value from stdin (one of them per call), keeping it out of
shell history and the process list.

Output (stdout): the saved fiscal information (secrets are reported only as
passwordSent / accessTokenSent / certificateSent booleans).`,
		Example: `  asaas fiscal-info save --email fiscal@empresa.com --simples-nacional=true \
    --municipal-inscription 123456 --cnae 6201501 --rps-serie 1 --rps-number 0

  printf '%s' "$CERT_PWD" | asaas fiscal-info save --email fiscal@empresa.com \
    --simples-nacional=true --certificate-file ./empresa.pfx --certificate-password -`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := cmdutil.CollectBody(cmd, saveFields)
			if err != nil {
				return err
			}
			if err := cmdutil.CheckRequired(body, saveFields); err != nil {
				return err
			}
			if err := readSecretsFromStdin(f, cmd, body); err != nil {
				return err
			}

			fields := make(map[string]string, len(body))
			for k, v := range body {
				fields[k] = fmt.Sprint(v)
			}

			var files []asaasclient.File
			if certFile != "" {
				fh, err := os.Open(certFile)
				if err != nil {
					return cmdutil.UsageErrorf("--certificate-file: %v", err)
				}
				defer fh.Close()
				files = append(files, asaasclient.File{Field: "certificateFile", Filename: filepath.Base(certFile), Content: fh})
			}

			client, err := f.Client()
			if err != nil {
				return err
			}
			resp, err := client.PostMultipart(cmd.Context(), basePath+"/", fields, files)
			if err != nil {
				return err
			}
			return f.PrintResponse(resp)
		},
	}
	cmdutil.AddFields(cmd, saveFields)
	cmd.Flags().StringVar(&certFile, "certificate-file", "", "Path to the digital certificate file (.pfx / .p12) for authenticationType CERTIFICATE [type: string, file path]")
	return cmd
}

func readSecretsFromStdin(f *cmdutil.Factory, cmd *cobra.Command, body map[string]any) error {
	var fromStdin []string
	for _, name := range secretFlags {
		if v, _ := cmd.Flags().GetString(name); cmd.Flags().Changed(name) && v == "-" {
			fromStdin = append(fromStdin, name)
		}
	}
	if len(fromStdin) == 0 {
		return nil
	}
	if len(fromStdin) > 1 {
		return cmdutil.UsageErrorf("only one secret can be read from stdin per call, got: --%s", strings.Join(fromStdin, ", --"))
	}
	line, err := bufio.NewReader(f.In).ReadString('\n')
	if err != nil && line == "" {
		return cmdutil.UsageErrorf("--%s -: could not read the value from stdin: %v", fromStdin[0], err)
	}
	for _, fl := range saveFields {
		if fl.Flag == fromStdin[0] {
			body[fl.Param] = strings.TrimRight(line, "\r\n")
		}
	}
	return nil
}
