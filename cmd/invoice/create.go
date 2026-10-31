package invoice

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

func createFields() []cmdutil.Field {
	return append([]cmdutil.Field{
		{Param: "payment", Flag: "payment", Usage: "Payment ID (pay_...) the invoice refers to; one of --payment, --installment or --customer is required"},
		{Param: "installment", Flag: "installment", Usage: "Installment ID the invoice refers to"},
		{Param: "customer", Flag: "customer", Usage: "Customer ID (cus_...) for a standalone invoice not tied to a payment"},
		{Param: "serviceDescription", Flag: "service-description", BodyRequired: true, Usage: "Description of the services rendered, printed on the invoice"},
		{Param: "observations", Flag: "observations", BodyRequired: true, Usage: "Additional notes printed on the invoice"},
		{Param: "value", Flag: "value", Type: cmdutil.Float, BodyRequired: true, Usage: "Total invoice amount in BRL (e.g. 300.00)"},
		{Param: "deductions", Flag: "deductions", Type: cmdutil.Float, Usage: "Deductions in BRL; lower the ISS base but not the total; defaults to 0 when omitted"},
		{Param: "effectiveDate", Flag: "effective-date", Type: cmdutil.Date, BodyRequired: true, Usage: "Issue date; the invoice is sent to the city hall on this date"},
		{Param: "municipalServiceId", Flag: "municipal-service-id", Usage: "Municipal service ID from 'asaas fiscal-info services'; one of --municipal-service-id or --municipal-service-code is required"},
		{Param: "municipalServiceCode", Flag: "municipal-service-code", Usage: "Municipal service code (required on the Portal Nacional, where no service list exists)"},
		{Param: "municipalServiceName", Flag: "municipal-service-name", Usage: "Municipal service name; when omitted Asaas uses the service code as the name"},
		{Param: "externalReference", Flag: "external-reference", Usage: "Your own identifier for this invoice"},
		{Param: "updatePayment", Flag: "update-payment", Type: cmdutil.Bool, Usage: "Reduce the linked payment's value by the withheld taxes"},
	}, taxFields(true)...)
}

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var data string
	fields := createFields()
	cmd := &cobra.Command{
		Use:   "create --service-description <text> --value <amount> --effective-date <YYYY-MM-DD> [flags]",
		Short: "Schedule a service invoice / NFS-e (POST /v3/invoices)",
		Long: `Create a service invoice and schedule it for --effective-date
(POST /v3/invoices). It is sent to the city hall / Portal Nacional on that
date; run 'asaas invoice authorize' to send it earlier.

Required (each via its flag or the same key in --data):
  - origin: one of --payment, --installment or --customer
  - service: one of --municipal-service-id or --municipal-service-code
  - --service-description, --observations, --value, --effective-date
  - taxes: --retain-iss, --iss, --pis, --cofins, --csll, --inss, --ir
    (rates in percent; use 0 for taxes that are not withheld)
--deductions defaults to 0. Missing fields are reported together as a
usage_error before any request is sent.

Real-estate services (tax reform) also need taxes.realEstateInfo and/or
taxes.referencedInvoiceAccessKeyList; pass them with --data.

Output (stdout): the created invoice object (status SCHEDULED), with its "id".`,
		Example: `  # Invoice for a received payment, Simples Nacional company, 2% ISS, no withholdings
  asaas invoice create --payment pay_080225913252 \
    --service-description "Consultoria em TI - outubro/2026" --observations "Mensal" \
    --value 300 --effective-date 2026-10-31 --municipal-service-id 60380 \
    --retain-iss=false --iss 2 --pis 0 --cofins 0 --csll 0 --inss 0 --ir 0

  # Standalone invoice on the Portal Nacional, taxes from a file
  asaas invoice create --customer cus_000005219613 --service-description "Desenvolvimento" \
    --observations "-" --value 1500 --effective-date 2026-11-05 \
    --municipal-service-code 01.07.01 --data @taxes.json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateOperationType(cmd); err != nil {
				return err
			}
			set, err := cmdutil.CollectBody(cmd, fields)
			if err != nil {
				return err
			}
			body, err := cmdutil.MergeJSONObject(f.In, "data", data, set)
			if err != nil {
				return err
			}
			if !cmdutil.HasPath(body, "deductions") {
				body["deductions"] = 0
			}
			if err := cmdutil.RequireOneOf(body, fields, "payment", "installment", "customer"); err != nil {
				return err
			}
			if err := cmdutil.RequireOneOf(body, fields, "municipalServiceId", "municipalServiceCode"); err != nil {
				return err
			}
			if err := cmdutil.CheckRequired(body, fields); err != nil {
				return err
			}
			return cmdutil.RunRequest(cmd, f, http.MethodPost, basePath, nil, body)
		},
	}
	cmdutil.AddFields(cmd, fields)
	cmd.Flags().StringVar(&data, "data", "", dataUsage)
	return cmd
}
