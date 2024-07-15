# GCLOUD Provider for DevPod

[![Join us on Slack!](docs/static/media/slack.svg)](https://slack.loft.sh/) [![Open in DevPod!](https://devpod.sh/assets/open-in-devpod.svg)](https://devpod.sh/open#https://github.com/loft-sh/devpod-provider-gcloud)

## Getting started

The provider is available for auto-installation using:

```sh
devpod provider add gcloud -o PROJECT=<project id to use> -o ZONE=<Google Cloud zone to create the VMs in>
devpod provider use gcloud
```

Option `PROJECT` must be set when adding the provider
(unless the project to be used is set as the current project in `gcloud`).

Option `ZONE` should be set when adding the provider.

Options can be set using `devpod provider set-options`, for example:

```sh
devpod provider set-options gcloud -o DISK_IMAGE=my-custom-vm-image
```

Be aware that authentication is obtained using `gcloud` CLI tool, take a look
[here](https://developers.google.com/accounts/docs/application-default-credentials)
for more information.

### Creating your first devpod workspace with gcloud

After the initial setup, just use:

```sh
devpod up .
```

You'll need to wait for the machine and workspace setup.

### Customize the VM Instance

This provides has the following options:

| NAME           | REQUIRED | DESCRIPTION                                                    | DEFAULT                                              |
|----------------|----------|----------------------------------------------------------------|------------------------------------------------------|
| DISK_IMAGE     | false    | The disk image to use.                                         | projects/cos-cloud/global/images/cos-101-17162-127-5 |
| DISK_SIZE      | false    | The disk size to use (GB).                                     | 40                                                   |
| MACHINE_TYPE   | false    | The machine type to use.                                       | c2-standard-4                                        |
| PROJECT        | true     | The project id to use.                                         |                                                      |
| ZONE           | true     | The google cloud zone to create the VM in. E.g. europe-west1-d |                                                      |
| NETWORK        | false    | The network id to use.                                         |                                                      |
| SUBNETWORK     | false    | The subnetwork id to use.                                      |                                                      |
| TAG            | false    | A tag to attach to the instance.                               | devpod                                               |
| SERVICE_ACCOUNT| false    | A service account to attach to instance


