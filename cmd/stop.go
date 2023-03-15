package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/gcloud"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/options"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
	"os"
)

// StopCmd holds the cmd flags
type StopCmd struct{}

// NewStopCmd defines a command
func NewStopCmd() *cobra.Command {
	cmd := &StopCmd{}
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			options, err := options.FromEnv(true)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), options, log.Default)
		},
	}

	return stopCmd
}

// Run runs the command logic
func (cmd *StopCmd) Run(ctx context.Context, options *options.Options, log log.Logger) error {
	clientOptions, err := getGCloudCredentials(ctx)
	if err != nil {
		return err
	}

	client, err := gcloud.NewClient(ctx, options.Project, options.Zone, clientOptions...)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Stop(ctx, options.MachineID, true)
}

func getGCloudCredentials(ctx context.Context) ([]option.ClientOption, error) {
	source, err := gcloud.DefaultTokenSource(ctx)
	if err != nil {
		providerToken := os.Getenv("GCLOUD_PROVIDER_TOKEN")
		if providerToken == "" {
			return nil, fmt.Errorf("couldn't find gcloud credentials")
		}

		source, err = gcloud.ParseToken(providerToken)
		if err != nil {
			return nil, err
		}
	}

	return []option.ClientOption{option.WithTokenSource(source)}, nil
}
