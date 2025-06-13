// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccExampleResource(t *testing.T) {
	source := os.Getenv("GCRANE_SOURCE")
	if source != "" {
		a := strings.Split(source, ":")
		randBytes := make([]byte, 16)
		_, err := rand.Read(randBytes)
		if err != nil {
			panic(err)
		}
		target := a[0] + ":" + hex.EncodeToString(randBytes)

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				// Create and Read testing
				{
					Config: testAccExampleResourceConfig(source, target),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(
							"gcrane_copy.copied_image",
							tfjsonpath.New("id"),
							knownvalue.StringExact(target),
						),
					},
				},
			},
		})
	}
}

func testAccExampleResourceConfig(source string, target string) string {
	return fmt.Sprintf(`
resource "gcrane_copy" "copied_image" {
  recursive = false

  source      = "%s"
  destination = "%s"
}
`, source, target)
}
