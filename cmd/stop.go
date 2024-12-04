package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/loft-sh/devpod-provider-gcloud/pkg/gcloud"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/options"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StopCmd holds the cmd flags
type StopCmd struct {
	Raw bool
}

// NewStopCmd defines a command
func NewStopCmd() *cobra.Command {
	cmd := &StopCmd{}
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			options, err := options.FromEnv(true, false)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), options, log.Default)
		},
	}

	stopCmd.Flags().BoolVar(&cmd.Raw, "raw", false, "If enabled will sent a raw request instead of using the SDK")
	return stopCmd
}

// Run runs the command logic
func (cmd *StopCmd) Run(ctx context.Context, options *options.Options, log log.Logger) error {
	if cmd.Raw {
		return rawStop(ctx, options)
	}

	client, err := gcloud.NewClient(ctx, options.Project, options.Zone)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Stop(ctx, options.MachineID, true)
}

func rawStop(ctx context.Context, options *options.Options) error {
	providerToken := os.Getenv("GCLOUD_PROVIDER_TOKEN")
	if providerToken == "" {
		return fmt.Errorf("couldn't find gcloud credentials")
	}

	tok, err := gcloud.ParseToken(providerToken)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s/stop", options.Project, options.Zone, options.MachineID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		out, err := io.ReadAll(resp.Body)
		if err == nil {
			return errors.Wrapf(err, "Error stopping vm: %s", string(out))
		}

		return err
	}

	return nil
}
