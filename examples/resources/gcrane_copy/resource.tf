resource "gcrane_copy" "copied_image" {
  recursive = false

  source      = "google/pause"
  destination = "europe-west4-docker.pkg.dev/my-project/my-repo/my-image:latest"
}
