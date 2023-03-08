package cmd

import (
	"cloud.google.com/go/compute/apiv1/computepb"
	"context"
	"fmt"
	"github.com/loft-sh/devpod-gcloud-provider/pkg/gcloud"
	"github.com/loft-sh/devpod-gcloud-provider/pkg/options"
	"github.com/loft-sh/devpod-gcloud-provider/pkg/ptr"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"strconv"
)

// CreateCmd holds the cmd flags
type CreateCmd struct{}

// NewCreateCmd defines a command
func NewCreateCmd() *cobra.Command {
	cmd := &CreateCmd{}
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			options, err := options.FromEnv()
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), options, log.Default)
		},
	}

	return createCmd
}

// Run runs the command logic
func (cmd *CreateCmd) Run(ctx context.Context, options *options.Options, log log.Logger) error {
	client, err := gcloud.NewClient(ctx, options.Project, options.Zone)
	if err != nil {
		return err
	}
	defer client.Close()

	instance, err := buildInstance(options)
	if err != nil {
		return err
	}

	return client.Create(ctx, instance)
}

func buildInstance(options *options.Options) (*computepb.Instance, error) {
	diskSize, err := strconv.Atoi(options.DiskSize)
	if err != nil {
		return nil, errors.Wrap(err, "parse disk size")
	}

	instance := &computepb.Instance{
		MachineType: ptr.Ptr(fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", options.Project, options.Zone, options.MachineType)),
		Disks: []*computepb.AttachedDisk{
			{
				AutoDelete: ptr.Ptr(true),
				Boot:       ptr.Ptr(true),
				DeviceName: ptr.Ptr(options.MachineID),
				InitializeParams: &computepb.AttachedDiskInitializeParams{
					DiskSizeGb:  ptr.Ptr(int64(diskSize)),
					DiskType:    ptr.Ptr(fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-balanced", options.Project, options.Zone)),
					SourceImage: ptr.Ptr(options.DiskImage),
				},
			},
		},
		NetworkInterfaces: []*computepb.NetworkInterface{
			{
				AccessConfigs: []*computepb.AccessConfig{
					{
						Name:        ptr.Ptr("External NAT"),
						NetworkTier: ptr.Ptr("STANDARD"),
					},
				},
			},
		},
		Zone: ptr.Ptr(fmt.Sprintf("projects/%s/zones/%s", options.Project, options.Zone)),
		Name: ptr.Ptr(options.MachineID),
	}

	return instance, nil
}
