package cmd

import (
	"os"
	"os/exec"

	"github.com/loft-sh/devpod-provider-dockerless/pkg/version"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// NewRootCmd returns a new root command
func NewRootCmd() *cobra.Command {
	dockerlessCmd := &cobra.Command{
		Use:           "devpod-provider-dockerless",
		Short:         "Dockerless Provider commands",
		SilenceErrors: true,
		SilenceUsage:  true,

		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			log.Default.MakeRaw()

			return nil
		},
	}

	return dockerlessCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// build the root command
	rootCmd := BuildRoot()

	// execute command
	err := rootCmd.Execute()
	if err != nil {
		//nolint: errorlint
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			if len(exitErr.Stderr) > 0 {
				log.Default.ErrorStreamOnly().Error(string(exitErr.Stderr))
			}

			os.Exit(exitErr.ExitCode())
		}
		log.Default.Fatal(err)
	}
}

// BuildRoot creates a new root command from the
func BuildRoot() *cobra.Command {
	rootCmd := NewRootCmd()
	rootCmd.Version = version.Version

	rootCmd.AddCommand(NewEnterCmd())
	rootCmd.AddCommand(NewFindCmd())
	rootCmd.AddCommand(NewDeleteCmd())
	rootCmd.AddCommand(NewStartCmd())
	rootCmd.AddCommand(NewRunCmd())
	rootCmd.AddCommand(NewCommandCmd())
	rootCmd.AddCommand(NewStopCmd())
	rootCmd.AddCommand(NewTargetArchitectureCmd())
	return rootCmd
}
