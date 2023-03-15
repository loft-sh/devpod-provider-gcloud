package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/gcloud"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/options"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/spf13/cobra"
)

// TokenCmd holds the cmd flags
type TokenCmd struct{}

// NewTokenCmd defines a command
func NewTokenCmd() *cobra.Command {
	cmd := &StopCmd{}
	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "Prints an access token",
		RunE: func(_ *cobra.Command, args []string) error {
			options, err := options.FromEnv()
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), options, log.Default)
		},
	}

	return tokenCmd
}

// Run runs the command logic
func (cmd *TokenCmd) Run(ctx context.Context, options *options.Options, log log.Logger) error {
	client, err := gcloud.NewClient(ctx, options.Project, options.Zone)
	if err != nil {
		return err
	}
	defer client.Close()

	tok, err := client.GetToken(ctx)
	if err != nil {
		return err
	}

	fmt.Print(string(tok))
	return nil
}
