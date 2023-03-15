package gcloud

import (
	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"context"
	"encoding/json"
	"fmt"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/loft-sh/devpod/pkg/client"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"strings"
)

func NewClient(ctx context.Context, project, zone string, opts ...option.ClientOption) (*Client, error) {
	instanceClient, err := compute.NewInstancesRESTClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		InstanceClient: instanceClient,
		Project:        project,
		Zone:           zone,
	}, nil
}

type Client struct {
	InstanceClient *compute.InstancesClient

	Project string
	Zone    string
}

func DefaultTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	return google.DefaultTokenSource(ctx, compute.DefaultAuthScopes()...)
}

func ParseToken(tok string) (oauth2.TokenSource, error) {
	oauthToken := &oauth2.Token{}
	err := json.Unmarshal([]byte(tok), oauthToken)
	if err != nil {
		return nil, err
	}

	return &token{tok: oauthToken}, nil
}

func GetToken(ctx context.Context) ([]byte, error) {
	tokSource, err := DefaultTokenSource(ctx)
	if err != nil {
		return nil, err
	}

	t, err := tokSource.Token()
	if err != nil {
		return nil, err
	}

	return json.Marshal(t)
}

func (c *Client) Init(ctx context.Context) error {
	_, err := c.InstanceClient.List(ctx, &computepb.ListInstancesRequest{
		Project: c.Project,
		Zone:    c.Zone,
	}).Next()
	if err != nil && err != iterator.Done {
		return fmt.Errorf("cannot list instances: %v", err)
	}

	return nil
}

func (c *Client) Create(ctx context.Context, instance *computepb.Instance) error {
	operation, err := c.InstanceClient.Insert(ctx, &computepb.InsertInstanceRequest{
		InstanceResource: instance,
		Project:          c.Project,
		Zone:             c.Zone,
	})
	if err != nil {
		return err
	}

	return operation.Wait(ctx)
}

func (c *Client) Start(ctx context.Context, name string) error {
	operation, err := c.InstanceClient.Start(ctx, &computepb.StartInstanceRequest{
		Instance: name,
		Project:  c.Project,
		Zone:     c.Zone,
	})
	if err != nil {
		return err
	}

	return operation.Wait(ctx)
}

func (c *Client) Stop(ctx context.Context, name string, async bool) error {
	operation, err := c.InstanceClient.Stop(ctx, &computepb.StopInstanceRequest{
		Instance: name,
		Project:  c.Project,
		Zone:     c.Zone,
	})
	if err != nil {
		return err
	} else if async {
		return nil
	}

	return operation.Wait(ctx)
}

func (c *Client) Delete(ctx context.Context, name string) error {
	operation, err := c.InstanceClient.Delete(ctx, &computepb.DeleteInstanceRequest{
		Instance: name,
		Project:  c.Project,
		Zone:     c.Zone,
	})
	if err != nil {
		return err
	}

	return operation.Wait(ctx)
}

func (c *Client) Get(ctx context.Context, name string) (*computepb.Instance, error) {
	instance, err := c.InstanceClient.Get(ctx, &computepb.GetInstanceRequest{
		Instance: name,
		Project:  c.Project,
		Zone:     c.Zone,
	})
	if err != nil {
		// check if api error
		apiError, ok := err.(*apierror.APIError)
		if ok {
			googleAPIError, ok := apiError.Unwrap().(*googleapi.Error)
			if ok && googleAPIError.Code == 404 {
				return nil, nil
			}
		}

		return nil, err
	}

	return instance, nil
}

func (c *Client) Status(ctx context.Context, name string) (client.Status, error) {
	instance, err := c.Get(ctx, name)
	if err != nil {
		return client.StatusNotFound, err
	} else if instance == nil {
		return client.StatusNotFound, nil
	}

	status := strings.TrimSpace(strings.ToUpper(*instance.Status))
	if status == "RUNNING" {
		return client.StatusRunning, nil
	} else if status == "STOPPING" || status == "SUSPENDING" || status == "REPAIRING" || status == "PROVISIONING" || status == "STAGING" {
		return client.StatusBusy, nil
	} else if status == "TERMINATED" {
		return client.StatusStopped, nil
	}

	return client.StatusNotFound, fmt.Errorf("unexpected status: %v", status)
}

func (c *Client) Close() error {
	err := c.InstanceClient.Close()
	if err != nil {
		return err
	}

	return nil
}

type token struct {
	tok *oauth2.Token
}

func (t *token) Token() (*oauth2.Token, error) {
	return t.tok, nil
}
