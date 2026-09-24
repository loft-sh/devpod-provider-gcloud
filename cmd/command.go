package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/loft-sh/devpod-provider-gcloud/pkg/gcloud"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/options"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/ssh"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
			options, err := options.FromEnv(true, true)
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
	privateKey, err := ssh.GetPrivateKeyRawBase(options.MachineFolder)
	if err != nil {
		return fmt.Errorf("load private key: %w", err)
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
	if options.PublicIP && (len(instance.NetworkInterfaces) == 0 || len(instance.NetworkInterfaces[0].AccessConfigs) == 0 || instance.NetworkInterfaces[0].AccessConfigs[0].NatIP == nil) {
		return fmt.Errorf("instance %s doesn't have an external nat ip", options.MachineID)
	}

	var target string
	var port string
	// get external address
	if options.PublicIP {
		target = *instance.NetworkInterfaces[0].AccessConfigs[0].NatIP
		port = "22"
	} else {
		// Pin to the IPv4 loopback explicitly everywhere below (tunnel bind,
		// readiness probe, and the SSH dial). "localhost" is ambiguous on
		// dual-stack hosts: gcloud's IAP tunnel can bind its two address
		// families at different times during startup, and if the readiness
		// probe and the final dial don't resolve "localhost" to the same
		// family, the dial can hit a family that isn't bound yet and fail
		// with "connection refused" even though the tunnel is otherwise up.
		target = "127.0.0.1"

		port, err = findAvailablePort()
		if err != nil {
			return err
		}

		// --project is required explicitly: this subprocess does not
		// necessarily inherit the same ambient gcloud config/env that an
		// interactive shell has, and without it gcloud falls back to
		// whatever project happens to be active in that context — which can
		// silently be the wrong one. A wrong project doesn't fail loudly;
		// IAP's backend can't find the instance there and the tunnel dies
		// with "Handshake status 404 Not Found (May be due to missing
		// permissions)", which then only shows up downstream as a plain
		// connection-refused on the SSH dial.
		gcloudIAPcommand := exec.Command("gcloud", []string{
			"compute",
			"start-iap-tunnel",
			*instance.Name,
			"22",
			"--local-host-port=" + target + ":" + port,
			// options.Zone (not *instance.Zone!): the Compute API's Instance.Zone
			// field is the full resource URL
			// (https://www.googleapis.com/compute/v1/projects/<p>/zones/<z>),
			// not a bare zone name. Passing that URL as --zone is invalid and
			// is exactly why IAP can't resolve the instance ("Handshake status
			// 404 Not Found (May be due to missing permissions)") on every
			// single attempt, regardless of timing or retries.
			"--zone=" + options.Zone,
			"--project=" + options.Project,
		}...)

		// Capture the tunnel's own output for diagnostics: if it fails to
		// bind/authenticate, we want to see why instead of just observing a
		// downstream "connection refused" on the dial below. devpod core does
		// not surface this provider's own log output through `devpod up
		// --debug`, so this is the only way to see it.
		tunnelLog, tunnelLogErr := os.CreateTemp("", "gcloud-iap-tunnel-*.log")
		if tunnelLogErr == nil {
			gcloudIAPcommand.Stdout = tunnelLog
			gcloudIAPcommand.Stderr = tunnelLog
			defer tunnelLog.Close()
			log.Debugf("iap tunnel log: %s", tunnelLog.Name())
		}

		// open tunnel in background
		if err = gcloudIAPcommand.Start(); err != nil {
			return fmt.Errorf("start tunnel: %w", err)
		}
		defer func() {
			err = gcloudIAPcommand.Process.Kill()
		}()

		timeoutCtx, cancelFn := context.WithTimeout(ctx, 30*time.Second)
		defer cancelFn()
		waitForPort(timeoutCtx, target, port)
	}

	// The tunnel can accept a TCP connection slightly before it's actually
	// proxying traffic (waitForPort only confirms the listening socket
	// exists), so retry the SSH dial a few times before giving up.
	dialDeadline := time.Now().Add(15 * time.Second)
	for {
		sshClient, dialErr := ssh.NewSSHClient("devpod", target+":"+port, privateKey)
		if dialErr == nil {
			defer sshClient.Close()
			return ssh.Run(ctx, sshClient, command, os.Stdin, os.Stdout, os.Stderr)
		}
		if time.Now().After(dialDeadline) {
			return errors.Wrap(dialErr, "create ssh client")
		}
		time.Sleep(1 * time.Second)
	}
}

func findAvailablePort() (string, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	defer l.Close()

	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

// waitForPort blocks until something is actively accepting connections on
// host:port, or ctx is done.
//
// The previous implementation polled readiness via net.Listen(host, port):
// success meant "still free, not ready yet", failure (address in use) meant
// "ready". That approach competes for the exact port the tunnel subprocess
// is trying to bind — if this probe's Listen call lands in the same instant
// the subprocess attempts its own bind, the subprocess can lose the race and
// fail to start, and this loop would happily report the port "free" again
// on the next iteration and eventually just time out, having killed the
// tunnel it was supposed to be waiting for. Dialing, instead of listening,
// never competes for the bind: it either connects (ready) or is refused
// (not ready yet).
func waitForPort(ctx context.Context, host, port string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := net.DialTimeout("tcp", host+":"+port, 500*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}
