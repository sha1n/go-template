package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version     string
	Build       string
	ProgramName string
)

func newRootCmd() *cobra.Command {
	programName := ProgramName
	if programName == "" {
		programName = "app"
	}

	rootCmd := &cobra.Command{
		Use:     programName,
		Short:   "A minimal Go CLI",
		Version: fmt.Sprintf("%s (build %s)", Version, Build),
		RunE: func(cmd *cobra.Command, _ []string) error {
			// cmd.Println writes to OutOrStderr, not OutOrStdout; the greeting
			// is normal output, so the writer is chosen explicitly.
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Lets Go!")

			return err
		},
	}

	// Explicit despite matching the zero value: with no other subcommands,
	// cobra only keeps "completion" registered when it is the command being
	// invoked (absent from --help, but `completion zsh` still works) unless
	// DisableDefaultCmd is set. Verified empirically against the built binary.
	rootCmd.CompletionOptions.DisableDefaultCmd = false

	return rootCmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
