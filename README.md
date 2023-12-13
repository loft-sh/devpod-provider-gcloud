# GCLOUD Provider for DevPod

[![Join us on Slack!](docs/static/media/slack.svg)](https://slack.loft.sh/) [![Open in DevPod!](https://devpod.sh/assets/open-in-devpod.svg)](https://devpod.sh/open#https://github.com/loft-sh/devpod-provider-gcloud)

## Getting started

The provider is available for auto-installation using 

```sh
devpod provider add gcloud
devpod provider use gcloud
```

Follow the on-screen instructions to complete the setup.

Needed variables will be:

- ZONE
- PROJECT

Be aware that authentication is obtained using `gcloud` CLI tool, take a look
[here](https://developers.google.com/accounts/docs/application-default-credentials)
for more info

### Creating your first devpod env with gcloud

After the initial setup, just use:

```sh
devpod up .
```

You'll need to wait for the machine and environment setup.

### Customize the VM Instance

This provides has the seguent options

| NAME           | REQUIRED | DESCRIPTION                                                    | DEFAULT                                              |
|----------------|----------|----------------------------------------------------------------|------------------------------------------------------|
| DISK_IMAGE     | false    | The disk image to use.                                         | projects/cos-cloud/global/images/cos-101-17162-127-5 |
| DISK_SIZE      | false    | The disk size to use.                                          | 40                                                   |
| MACHINE_TYPE   | false    | The machine type to use.                                       | c2-standard-4                                        |
| PROJECT        | true     | The project id to use.                                         |                                                      |
| ZONE           | true     | The google cloud zone to create the VM in. E.g. europe-west1-d |                                                      |
| NETWORK        | false    | The network id to use.                                         |                                                      |
| SUBNETWORK     | false    | The subnetwork id to use.                                      |                                                      |
| TAG            | false    | A tag to attach to the instance.                               | devpod                                               |
| SERVICE_ACCOUNT| false    | A service account to attach to instance

Options can either be set in `env` or using for example:

```sh
devpod provider set-options -o DISK_IMAGE=my-custom-vm-image
```
