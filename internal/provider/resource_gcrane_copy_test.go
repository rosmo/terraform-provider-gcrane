// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
