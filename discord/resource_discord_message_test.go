package discord

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDiscordMessageContent(t *testing.T) {
	testChannelID := os.Getenv("DISCORD_TEST_CHANNEL_ID")
	if testChannelID == "" {
		t.Skip("DISCORD_TEST_CHANNEL_ID envvar must be set for acceptance tests")
	}
	name := "discord_message.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordMessageContent(testChannelID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "channel_id", testChannelID),
					resource.TestCheckResourceAttr(name, "content", "Hello, World from Terraform!"),
					resource.TestCheckResourceAttr(name, "tts", "false"),
				),
			},
		},
	})
}

func TestAccResourceDiscordMessageEmbed(t *testing.T) {
	testChannelID := os.Getenv("DISCORD_TEST_CHANNEL_ID")
	if testChannelID == "" {
		t.Skip("DISCORD_TEST_CHANNEL_ID envvar must be set for acceptance tests")
	}
	name := "discord_message.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordEmbed(testChannelID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "channel_id", testChannelID),
					resource.TestCheckResourceAttr(name, "embed.#", "1"),
					resource.TestCheckResourceAttr(name, "embed.0.title", "Hello, World from Terraform!"),
					resource.TestCheckResourceAttr(name, "embed.0.description", "This is a test message from Terraform!"),
					resource.TestCheckResourceAttr(name, "embed.0.color", "65280"),
					resource.TestCheckResourceAttr(name, "embed.0.footer.0.text", "This is a test footer from Terraform!"),
				),
			},
		},
	})
}

func TestAccResourceDiscordMessageFile(t *testing.T) {
	testChannelID := os.Getenv("DISCORD_TEST_CHANNEL_ID")
	if testChannelID == "" {
		t.Skip("DISCORD_TEST_CHANNEL_ID envvar must be set for acceptance tests")
	}

	dir := t.TempDir()
	imagePath := filepath.Join(dir, "hello.png")
	// Smallest valid 1x1 transparent PNG.
	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(imagePath, pngBytes, 0o644); err != nil {
		t.Fatalf("seed image: %v", err)
	}

	name := "discord_message.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordMessageFile(testChannelID, imagePath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "channel_id", testChannelID),
					resource.TestCheckResourceAttr(name, "content", "Hello from Terraform with attachment!"),
					resource.TestCheckResourceAttr(name, "file.#", "1"),
					resource.TestCheckResourceAttr(name, "file.0.filename", "hello.png"),
					resource.TestCheckResourceAttrSet(name, "file.0.id"),
					resource.TestCheckResourceAttrSet(name, "file.0.url"),
					resource.TestCheckResourceAttrSet(name, "file.0.size"),
				),
			},
		},
	})
}

func testAccResourceDiscordMessageFile(channelID, imagePath string) string {
	return fmt.Sprintf(`
	resource "discord_message" "example" {
      channel_id = "%[1]s"
      content    = "Hello from Terraform with attachment!"
      file {
        source = "%[2]s"
      }
	}`, channelID, imagePath)
}

func testAccResourceDiscordMessageContent(channelID string) string {
	return fmt.Sprintf(`
	resource "discord_message" "example" {
      channel_id = "%[1]s"
      content = "Hello, World from Terraform!"
	  tts = false
	}`, channelID)
}

func testAccResourceDiscordEmbed(channelID string) string {
	return fmt.Sprintf(`
    data "discord_color" "green" {
    	hex = "#00ff00"
		}
	resource "discord_message" "example" {
      channel_id = "%[1]s"
      embed {
			title = "Hello, World from Terraform!"
            description = "This is a test message from Terraform!"
 		   color = data.discord_color.green.dec
 		   footer {
              text = "This is a test footer from Terraform!"
		   }
		}
	}`, channelID)
}
