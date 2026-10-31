// Package invoice implements "asaas invoice": service invoices (NFS-e)
// through /v3/invoices.
package invoice

import (
	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

const basePath = "/invoices"

// Statuses accepted by the GET /v3/invoices "status" filter.
var statuses = []string{"SCHEDULED", "AUTHORIZED", "PROCESSING_CANCELLATION", "CANCELED", "CANCELLATION_DENIED", "ERROR"}

// NewCmdInvoice builds the "invoice" command group.
func NewCmdInvoice(f *cmdutil.Factory) *cobra.Command {
	cmd := cmdutil.NewGroup("invoice <subcommand>",
		"Manage service invoices / NFS-e (/v3/invoices): list, get, create, update, authorize, cancel",
		`Manage service invoices (Nota Fiscal de Serviço eletrônica, NFS-e).

Lifecycle:
  1. 'invoice create' schedules an invoice for its effectiveDate (status SCHEDULED).
  2. On that date Asaas sends it to the city hall / Portal Nacional; use
     'invoice authorize' to send a scheduled invoice right away.
  3. The invoice becomes AUTHORIZED (or ERROR). Authorization is asynchronous:
     poll 'invoice get' or use the INVOICE_AUTHORIZED / INVOICE_ERROR webhooks.
  4. 'invoice cancel' requests cancellation (PROCESSING_CANCELLATION, then
     CANCELED or CANCELLATION_DENIED).

Prerequisite: the account's fiscal information must be configured
(see 'asaas fiscal-info'). Use 'asaas fiscal-info services' to find the
municipalServiceId, or pass --municipal-service-code when the city has no
service list (always the case on the Portal Nacional).

Invoice IDs look like "inv_000000000232". An invoice object contains, among
others: id, status, customer, payment, installment, type, statusDescription,
serviceDescription, pdfUrl, xmlUrl, rpsSerie, rpsNumber, number,
validationCode, value, deductions, effectiveDate, observations,
estimatedTaxesDescription, externalReference, taxes, municipalServiceId,
municipalServiceCode, municipalServiceName.`)

	cmd.AddCommand(
		newCmdList(f),
		newCmdGet(f),
		newCmdCreate(f),
		newCmdUpdate(f),
		newCmdAuthorize(f),
		newCmdCancel(f),
	)
	return cmd
}

// taxRequired are the keys Asaas requires whenever a "taxes" object is sent.
var taxRequired = []string{"retainIss", "iss", "pis", "cofins", "csll", "inss", "ir"}

// taxFields map to the nested "taxes" object. Rates are percentages
// (2.5 means 2.5%). required controls whether the mandatory ones must be
// present in the final body (create) or not (update).
func taxFields(required bool) []cmdutil.Field {
	return []cmdutil.Field{
		{Param: "taxes.retainIss", Flag: "retain-iss", Type: cmdutil.Bool, BodyRequired: required, Usage: "Whether the customer (tomador) withholds the ISS; pass --retain-iss=false explicitly when not"},
		{Param: "taxes.iss", Flag: "iss", Type: cmdutil.Float, BodyRequired: required, Usage: "ISS rate in percent (e.g. 2.5 for 2.5%)"},
		{Param: "taxes.pis", Flag: "pis", Type: cmdutil.Float, BodyRequired: required, Usage: "PIS withholding rate in percent; 0 when not withheld"},
		{Param: "taxes.cofins", Flag: "cofins", Type: cmdutil.Float, BodyRequired: required, Usage: "COFINS withholding rate in percent; 0 when not withheld"},
		{Param: "taxes.csll", Flag: "csll", Type: cmdutil.Float, BodyRequired: required, Usage: "CSLL withholding rate in percent; 0 when not withheld"},
		{Param: "taxes.inss", Flag: "inss", Type: cmdutil.Float, BodyRequired: required, Usage: "INSS withholding rate in percent; 0 when not withheld"},
		{Param: "taxes.ir", Flag: "ir", Type: cmdutil.Float, BodyRequired: required, Usage: "IR withholding rate in percent; 0 when not withheld"},
		{Param: "taxes.nbsCode", Flag: "nbs-code", Usage: "NBS code (Nomenclatura Brasileira de Serviços); see 'asaas fiscal-info nbs-codes'"},
		{Param: "taxes.taxSituationCode", Flag: "tax-situation-code", Usage: "Tax situation code (código de situação tributária, tax reform)"},
		{Param: "taxes.taxClassificationCode", Flag: "tax-classification-code", Usage: "Tax classification code (código de classificação tributária, tax reform)"},
		{Param: "taxes.operationIndicatorCode", Flag: "operation-indicator-code", Usage: "Operation indicator code (código do indicador de operação, tax reform), e.g. 020101"},
		{Param: "taxes.pisCofinsTaxStatus", Flag: "pis-cofins-tax-status", Usage: "PIS/COFINS tax status (situação tributária do PIS/COFINS)"},
		{Param: "taxes.operationPis", Flag: "operation-pis", Type: cmdutil.Float, Usage: "Operation PIS rate in percent (contributor's own assessment)"},
		{Param: "taxes.operationCofins", Flag: "operation-cofins", Type: cmdutil.Float, Usage: "Operation COFINS rate in percent (contributor's own assessment)"},
		{Param: "taxes.operationTypeCode", Flag: "operation-type-code", Type: cmdutil.Int, Usage: "Tax-reform operation type, 1-5; only for real-estate services (federal tax codes 25.05, 15.09, 17.12, 10.05)"},
	}
}

const dataUsage = `Extra request body fields as a JSON object: inline, @file.json or - for stdin. Deep-merged under the typed flags (flags win). Use it for taxes.realEstateInfo and taxes.referencedInvoiceAccessKeyList, e.g. '{"taxes":{"referencedInvoiceAccessKeyList":["<50 digits>"]}}' [type: JSON object]`

func idArg(_ *cobra.Command, args []string) error {
	if len(args) != 1 || args[0] == "" {
		return cmdutil.UsageErrorf("expected exactly one <invoice-id> argument (e.g. inv_000000000232), got %d", len(args))
	}
	return nil
}

// validateOperationType checks taxes.operationTypeCode bounds when set by flag.
func validateOperationType(cmd *cobra.Command) error {
	if cmd.Flags().Changed("operation-type-code") {
		if v, _ := cmd.Flags().GetInt("operation-type-code"); v < 1 || v > 5 {
			return cmdutil.UsageErrorf("--operation-type-code must be between 1 and 5, got %d", v)
		}
	}
	return nil
}
