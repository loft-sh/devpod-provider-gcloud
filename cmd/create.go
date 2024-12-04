package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/gcloud"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/options"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/ptr"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/ssh"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
			options, err := options.FromEnv(true, true)
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

	// generate ssh keys
	publicKeyBase, err := ssh.GetPublicKeyBase(options.MachineFolder)
	if err != nil {
		return nil, errors.Wrap(err, "generate public key")
	}

	publicKey, err := base64.StdEncoding.DecodeString(publicKeyBase)
	if err != nil {
		return nil, err
	}
	serviceAccounts := []*computepb.ServiceAccount{}
	if options.ServiceAccount != "" {
		serviceAccounts = []*computepb.ServiceAccount{
			{
				Email: &options.ServiceAccount,
				Scopes: []string{
					"https://www.googleapis.com/auth/cloud-platform",
				},
			},
		}
	}

	// generate instance object
	instance := &computepb.Instance{
		Scheduling: &computepb.Scheduling{
			AutomaticRestart:  ptr.Ptr(true),
			OnHostMaintenance: ptr.Ptr(getMaintenancePolicy(options.MachineType)),
		},
		Metadata: &computepb.Metadata{
			Items: []*computepb.Items{
				{
					Key:   ptr.Ptr("ssh-keys"),
					Value: ptr.Ptr("devpod:" + string(publicKey)),
				},
			},
		},
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
		Tags: buildInstanceTags(options),
		NetworkInterfaces: []*computepb.NetworkInterface{
			{
				Network:       normalizeNetworkID(options),
				Subnetwork:    normalizeSubnetworkID(options),
				AccessConfigs: getAccessConfig(options),
			},
		},
		Zone:            ptr.Ptr(fmt.Sprintf("projects/%s/zones/%s", options.Project, options.Zone)),
		Name:            ptr.Ptr(options.MachineID),
		ServiceAccounts: serviceAccounts,
	}

	return instance, nil
}

func getAccessConfig(options *options.Options) []*computepb.AccessConfig {
	if options.PublicIP {
		return []*computepb.AccessConfig{
			{
				Name:        ptr.Ptr("External NAT"),
				NetworkTier: ptr.Ptr("STANDARD"),
			},
		}
	}

	return nil
}

func buildInstanceTags(options *options.Options) *computepb.Tags {
	if len(options.Tag) == 0 {
		return nil
	}

	return &computepb.Tags{Items: []string{options.Tag}}
}

func normalizeNetworkID(options *options.Options) *string {
	network := options.Network
	project := options.Project

	if len(network) == 0 {
		return nil
	}

	// projects/{{project}}/regions/{{region}}/subnetworks/{{name}}
	if regexp.MustCompile("projects/([^/]+)/global/networks/([^/]+)").MatchString(network) {
		return ptr.Ptr(network)
	}

	// {{project}}/{{name}}
	if regexp.MustCompile("([^/]+)/([^/]+)").MatchString(network) {
		s := strings.Split(network, "/")
		return ptr.Ptr(fmt.Sprintf("projects/%s/global/networks/%s", s[0], s[1]))
	}

	// {{name}}
	return ptr.Ptr(fmt.Sprintf("projects/%s/global/networks/%s", project, network))
}

func normalizeSubnetworkID(options *options.Options) *string {
	sn := strings.TrimSpace(options.Subnetwork)

	if len(sn) == 0 {
		return nil
	}

	project := options.Project
	zone := options.Zone
	region := zone[:strings.LastIndex(zone, "-")]

	// projects/{{project}}/regions/{{region}}/subnetworks/{{name}}
	if regexp.MustCompile("projects/([^/]+)/regions/([^/]+)/subnetworks/([^/]+)").MatchString(sn) {
		return ptr.Ptr(sn)
	}

	// {{project}}/{{region}}/{{name}}
	if regexp.MustCompile("([^/]+)/([^/]+)/([^/]+)").MatchString(sn) {
		s := strings.Split(sn, "/")
		return ptr.Ptr(fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", s[0], s[1], s[2]))
	}

	// {{region}}/{{name}}
	if regexp.MustCompile("([^/]+)/([^/]+)").MatchString(sn) {
		s := strings.Split(sn, "/")
		return ptr.Ptr(fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", project, s[0], s[1]))
	}

	// {{name}}
	return ptr.Ptr(fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", project, region, sn))
}

var gpuInstancePattern *regexp.Regexp = regexp.MustCompile(`^[agn][0-9]`)

func getMaintenancePolicy(machineType string) string {
	if gpuInstancePattern.MatchString(machineType) {
		return "TERMINATE"
	}

	return "MIGRATE"
}
