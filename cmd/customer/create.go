package customer

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var data string
	fields := bodyFields(true)
	cmd := &cobra.Command{
		Use:   "create --name <name> --cpf-cnpj <digits> [flags]",
		Short: "Create a customer (POST /v3/customers)",
		Long: `Create a customer (POST /v3/customers). Only flags that are explicitly set
are sent. Fields without a dedicated flag can be passed with --data.

Note: Asaas does not prevent duplicates; run 'asaas customer list --cpf-cnpj'
first if a customer with the same document may already exist.

Output (stdout): the created customer object, including its new "id".`,
		Example: `  asaas customer create --name "Marcelo Almeida" --cpf-cnpj 24971563792
  asaas customer create --name "ACME Ltda" --cpf-cnpj 12345678000195 \
    --email financeiro@acme.com --mobile-phone 47999376637 --postal-code 01310000
  asaas customer create --name "Ana" --cpf-cnpj 24971563792 --data @extra.json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			set, err := cmdutil.CollectBody(cmd, fields)
			if err != nil {
				return err
			}
			body, err := cmdutil.MergeJSONObject(f.In, "data", data, set)
			if err != nil {
				return err
			}
			return cmdutil.RunRequest(cmd, f, http.MethodPost, basePath, nil, body)
		},
	}
	cmdutil.AddFields(cmd, fields)
	cmd.Flags().StringVar(&data, "data", "", dataUsage)
	return cmd
}
