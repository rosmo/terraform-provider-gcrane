# Terraform Provider for gcrane

Terraform provider for [gcrane](https://github.com/google/go-containerregistry/blob/main/cmd/gcrane/README.md).

Allows copying images between Docker registries and also fetching some details (like images, tags, etc).
Does not require `gcrane` or Docker installed.

This is a
[community maintained provider](https://www.terraform.io/docs/providers/type/community-index.html)
and not an official Google or Hashicorp product.


```hcl
# The provider creates a temporary Docker config
provider "gcrane" {
  docker_config = <<-EOT
    {
      "auths": {
        "https://index.docker.io/v1/": {
          "auth": "12345678...abc..."
        },
      }
      "credHelpers": {
        "europe-west4-docker.pkg.dev": "gcloud"
      }
    }
  EOT
}

resource "gcrane_copy" "copied_image" {
  recursive = false

  source = "google/cloud-sdk:slim"
  destination = "europe-west4-docker.pkg.dev/my-project/my-repo/my-image:latest"
}

data "gcrane_list" "images" {
  repository = "google/cloud-sdk"
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


