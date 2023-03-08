package cmd

import (
	log2 "github.com/loft-sh/devpod/pkg/log"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"os"
	"os/exec"
)

// NewRootCmd returns a new root command
func NewRootCmd() *cobra.Command {
	gcloudCmd := &cobra.Command{
		Use:   "devpod-provider-gcloud",
		Short: "gcloud Provider commands",

		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			log2.Default.MakeRaw()
			return nil
		},
	}

	return gcloudCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// build the root command
	rootCmd := BuildRoot()

	// execute command
	err := rootCmd.Execute()
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			os.Exit(exitErr.ExitStatus())
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if len(exitErr.Stderr) > 0 {
				log2.Default.ErrorStreamOnly().Error(string(exitErr.Stderr))
			}
			os.Exit(exitErr.ExitCode())
		}

		os.Exit(1)
	}
}

// BuildRoot creates a new root command from the
func BuildRoot() *cobra.Command {
	rootCmd := NewRootCmd()

	rootCmd.AddCommand(NewCreateCmd())
	rootCmd.AddCommand(NewStatusCmd())
	rootCmd.AddCommand(NewDeleteCmd())
	rootCmd.AddCommand(NewStartCmd())
	rootCmd.AddCommand(NewStopCmd())
	return rootCmd
}
