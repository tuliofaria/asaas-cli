package cmd

import (
	"runtime"

	"github.com/spf13/cobra"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
)

func newCmdVersion(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version as JSON",
		Long: `Print build information as JSON:
  {"version": "<semver or dev>", "go_version": "...", "os": "...", "arch": "..."}`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return f.Print(map[string]string{
				"version":    f.Version,
				"go_version": runtime.Version(),
				"os":         runtime.GOOS,
				"arch":       runtime.GOARCH,
			})
		},
	}
}
