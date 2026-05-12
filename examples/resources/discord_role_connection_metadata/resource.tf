resource "discord_role_connection_metadata" "example" {
  application_id = "123456789012345678"

  metadata {
    key         = "verified"
    name        = "Verified"
    type        = 7
    description = "Has verified their account"
  }

  metadata {
    key         = "member_since"
    name        = "Member Since"
    type        = 6
    description = "Date the user became a member"
  }
}
