package cmdutil

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

// NewGroup builds a parent command (e.g. "customer") that only hosts
// subcommands. Running it alone prints help; an unknown subcommand is a
// usage error.
func NewGroup(use, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
}

// ResourcePath joins a collection path with escaped path segments,
// e.g. ResourcePath("/customers", id, "restore").
func ResourcePath(base string, segments ...string) string {
	parts := []string{strings.TrimRight(base, "/")}
	for _, s := range segments {
		parts = append(parts, url.PathEscape(s))
	}
	return strings.Join(parts, "/")
}

// RunList runs a GET on a list endpoint using filters declared as Fields.
// With all=true every page is fetched and merged into one list.
func RunList(cmd *cobra.Command, f *Factory, path string, filters []Field, all bool) error {
	if err := ValidatePagination(cmd); err != nil {
		return err
	}
	query, err := CollectQuery(cmd, filters)
	if err != nil {
		return err
	}
	client, err := f.Client()
	if err != nil {
		return err
	}
	if all {
		page, err := client.ListAll(cmd.Context(), path, query)
		if err != nil {
			return err
		}
		return f.Print(page)
	}
	resp, err := client.Get(cmd.Context(), path, query)
	if err != nil {
		return err
	}
	return f.PrintResponse(resp)
}

// RunRequest sends one request with the resolved client and prints the response.
func RunRequest(cmd *cobra.Command, f *Factory, method, path string, query url.Values, body any) error {
	client, err := f.Client()
	if err != nil {
		return err
	}
	resp, err := client.Do(cmd.Context(), method, path, query, body)
	if err != nil {
		return err
	}
	return f.PrintResponse(resp)
}
