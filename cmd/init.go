package cmd

import (
	"context"

	"github.com/loft-sh/devpod-provider-gcloud/pkg/gcloud"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/options"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/spf13/cobra"
)

// InitCmd holds the cmd flags
type InitCmd struct{}

// NewInitCmd defines a command
func NewInitCmd() *cobra.Command {
	cmd := &InitCmd{}
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Init an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			options, err := options.FromEnv(false, false)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), options, log.Default)
		},
	}

	return initCmd
}

// Run runs the command logic
func (cmd *InitCmd) Run(ctx context.Context, options *options.Options, log log.Logger) error {
	client, err := gcloud.NewClient(ctx, options.Project, options.Zone)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Init(ctx)
}
