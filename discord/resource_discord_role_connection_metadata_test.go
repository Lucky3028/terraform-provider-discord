package discord

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDiscordRoleConnectionMetadata(t *testing.T) {
	testAppID := os.Getenv("DISCORD_TEST_APPLICATION_ID")
	if testAppID == "" {
		t.Skip("DISCORD_TEST_APPLICATION_ID envvar must be set for acceptance tests")
	}
	name := "discord_role_connection_metadata.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordRoleConnectionMetadata(testAppID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "application_id", testAppID),
					resource.TestCheckResourceAttr(name, "metadata.#", "1"),
					resource.TestCheckResourceAttr(name, "metadata.0.key", "verified"),
					resource.TestCheckResourceAttr(name, "metadata.0.name", "Verified"),
					resource.TestCheckResourceAttr(name, "metadata.0.type", "7"),
					resource.TestCheckResourceAttr(name, "metadata.0.description", "Has verified their account"),
				),
			},
		},
	})
}

func testAccResourceDiscordRoleConnectionMetadata(appID string) string {
	return fmt.Sprintf(`
	resource "discord_role_connection_metadata" "example" {
	  application_id = "%[1]s"

	  metadata {
	    key         = "verified"
	    name        = "Verified"
	    type        = 7
	    description = "Has verified their account"
	  }
	}`, appID)
}
