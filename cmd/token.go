package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/gcloud"
	"github.com/spf13/cobra"
)

// TokenCmd holds the cmd flags
type TokenCmd struct{}

// NewTokenCmd defines a command
func NewTokenCmd() *cobra.Command {
	cmd := &TokenCmd{}
	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "Prints an access token",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background())
		},
	}

	return tokenCmd
}

// Run runs the command logic
func (cmd *TokenCmd) Run(ctx context.Context) error {
	tok, err := gcloud.GetToken(ctx)
	if err != nil {
		return err
	}

	fmt.Print(string(tok))
	return nil
}
