// Package customer implements "asaas customer": the /v3/customers resource.
package customer

import (
	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

const basePath = "/customers"

// NewCmdCustomer builds the "customer" command group.
func NewCmdCustomer(f *cmdutil.Factory) *cobra.Command {
	cmd := cmdutil.NewGroup("customer <subcommand>",
		"Manage customers (/v3/customers): list, get, create, update, delete, restore",
		`Manage the customers (payers) of the authenticated Asaas account.
Every subcommand calls /v3/customers and prints the raw Asaas JSON response.

Customer IDs look like "cus_000005219613". A customer object contains, among
others: id, name, cpfCnpj, personType (FISICA|JURIDICA), email, phone,
mobilePhone, address, addressNumber, complement, province, postalCode,
city, state, externalReference, notificationDisabled, additionalEmails,
groupName, company, foreignCustomer, deleted, dateCreated.`)

	cmd.AddCommand(
		newCmdList(f),
		newCmdGet(f),
		newCmdCreate(f),
		newCmdUpdate(f),
		newCmdDelete(f),
		newCmdRestore(f),
	)
	return cmd
}

// bodyFields are the writable customer attributes shared by create and update.
func bodyFields(requireIdentity bool) []cmdutil.Field {
	return []cmdutil.Field{
		{Param: "name", Flag: "name", Usage: "Customer full name or company name", Required: requireIdentity},
		{Param: "cpfCnpj", Flag: "cpf-cnpj", Usage: "CPF (11 digits) or CNPJ (14 digits), digits only", Required: requireIdentity},
		{Param: "email", Flag: "email", Usage: "Primary e-mail address"},
		{Param: "phone", Flag: "phone", Usage: "Landline phone with area code, digits only (e.g. 4738010919)"},
		{Param: "mobilePhone", Flag: "mobile-phone", Usage: "Mobile phone with area code, digits only (e.g. 47999376637)"},
		{Param: "address", Flag: "address", Usage: "Street name"},
		{Param: "addressNumber", Flag: "address-number", Usage: "Street number"},
		{Param: "complement", Flag: "complement", Usage: "Address complement (apartment, suite...)"},
		{Param: "province", Flag: "province", Usage: "Neighborhood (bairro)"},
		{Param: "postalCode", Flag: "postal-code", Usage: "Brazilian postal code (CEP), 8 digits"},
		{Param: "externalReference", Flag: "external-reference", Usage: "Your own identifier for this customer"},
		{Param: "notificationDisabled", Flag: "notification-disabled", Type: cmdutil.Bool, Usage: "Disable Asaas e-mail/SMS notifications to this customer"},
		{Param: "additionalEmails", Flag: "additional-emails", Usage: "Extra e-mails that receive notifications, comma-separated (e.g. a@x.com,b@y.com)"},
		{Param: "municipalInscription", Flag: "municipal-inscription", Usage: "Municipal registration number (inscrição municipal)"},
		{Param: "stateInscription", Flag: "state-inscription", Usage: "State registration number (inscrição estadual)"},
		{Param: "observations", Flag: "observations", Usage: "Free-text notes"},
		{Param: "groupName", Flag: "group-name", Usage: "Customer group name"},
		{Param: "company", Flag: "company", Usage: "Company name (for individuals linked to a company)"},
		{Param: "foreignCustomer", Flag: "foreign-customer", Type: cmdutil.Bool, Usage: "Customer is not Brazilian (no CPF/CNPJ validation)"},
	}
}

const dataUsage = `Extra request body fields as a JSON object: inline, @file.json or - for stdin. Typed flags override keys present here (e.g. '{"company":"ACME"}') [type: JSON object]`

func idArg(cmd *cobra.Command, args []string) error {
	if len(args) != 1 || args[0] == "" {
		return cmdutil.UsageErrorf("expected exactly one <customer-id> argument (e.g. cus_000005219613), got %d", len(args))
	}
	return nil
}
