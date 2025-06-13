# Terraform Provider for gcrane

Terraform provider for [gcrane](https://github.com/google/go-containerregistry/blob/main/cmd/gcrane/README.md).

```hcl
# The provider creates a temporary Docker config
provider "gcrane" {
  docker_config = <<-EOT
    {
        credHelpers = {
            "europe-west4-docker.pkg.dev" = "gcloud"
        }
    }
  EOT
}

resource "gcrane_copy" "copied_image" {
    recursive = false

    source = "artifactory.net/foo"
    destination = "europe-west4-docker.pkg.dev/my-project/my-repo/my-image:latest"
}

data "gcrane_list" "images" {
    repository = "artifactory.net/foo"
}
```

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Fill this in for each provider

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
