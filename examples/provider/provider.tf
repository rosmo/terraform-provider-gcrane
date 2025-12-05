provider "gcrane" {
  docker_config = <<-EOT
      {
        "auths": {
          "your.repository.example.com": {
            "auth": "your-token-here"
          }
        },
        "credHelpers": {
          "europe-docker.pkg.dev": "gcloud"
        }
      }
    EOT
}