resource "discord_message" "instructions_step_1" {
  channel_id = var.channel_id
  content    = "**Step 1.** Click the arrow next to the server name."

  file {
    source = "${path.module}/assets/step1.png"
  }
}

resource "discord_message" "release_notes" {
  channel_id = var.channel_id
  content    = "New release! See the attached changelog and screenshot."

  file {
    source   = "${path.module}/assets/changelog.txt"
    filename = "CHANGELOG-v1.2.0.txt"
  }

  file {
    source = "${path.module}/assets/screenshot.png"
  }
}
