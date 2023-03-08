package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/gcloud"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/options"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/log"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

// CommandCmd holds the cmd flags
type CommandCmd struct{}

// NewCommandCmd defines a command
func NewCommandCmd() *cobra.Command {
	cmd := &CommandCmd{}
	commandCmd := &cobra.Command{
		Use:   "command",
		Short: "Run a command on the instance",
		RunE: func(_ *cobra.Command, args []string) error {
			options, err := options.FromEnv()
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), options, log.Default)
		},
	}

	return commandCmd
}

// Run runs the command logic
func (cmd *CommandCmd) Run(ctx context.Context, options *options.Options, log log.Logger) error {
	command := os.Getenv("COMMAND")
	if command == "" {
		return fmt.Errorf("command environment variable is missing")
	}

	// get private key
	privateKey, err := ssh.GetPrivateKey(options.MachineFolder)
	if err != nil {
		return fmt.Errorf("load private key: %v", err)
	}

	// create gcloud client
	client, err := gcloud.NewClient(ctx, options.Project, options.Zone)
	if err != nil {
		return err
	}
	defer client.Close()

	// get instance
	instance, err := client.Get(ctx, options.MachineID)
	if err != nil {
		return err
	} else if instance == nil {
		return fmt.Errorf("instance %s doesn't exist", options.MachineID)
	}

	// get external ip
	if len(instance.NetworkInterfaces) == 0 || len(instance.NetworkInterfaces[0].AccessConfigs) == 0 || instance.NetworkInterfaces[0].AccessConfigs[0].NatIP == nil {
		return fmt.Errorf("instance %s doesn't have an external nat ip", options.MachineID)
	}

	// get external address
	externalIP := *instance.NetworkInterfaces[0].AccessConfigs[0].NatIP
	sshClient, err := ssh.NewClient(externalIP+":22", []byte(privateKey))
	if err != nil {
		return errors.Wrap(err, "create ssh client")
	}
	defer sshClient.Close()

	// run command
	return devssh.Run(sshClient, command, os.Stdin, os.Stdout, os.Stderr)
}
