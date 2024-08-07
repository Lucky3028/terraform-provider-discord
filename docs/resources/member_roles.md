---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "discord_member_roles Resource - discord"
subcategory: ""
description: |-
  A resource to manage member roles for a server.
---

# discord_member_roles (Resource)

A resource to manage member roles for a server.

## Example Usage

```terraform
resource "discord_member_roles" "jake" {
  user_id   = var.user_id
  server_id = var.server_id

  role {
    role_id = var.role_id_to_add
  }

  role {
    role_id  = var.role_id_to_always_remove
    has_role = false
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `role` (Block Set, Min: 1) Roles to manage. (see [below for nested schema](#nestedblock--role))
- `server_id` (String) ID of the server to manage roles in.
- `user_id` (String) ID of the user to manage roles for.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--role"></a>
### Nested Schema for `role`

Required:

- `role_id` (String) The role ID to manage.

Optional:

- `has_role` (Boolean) Whether the user should have the role. (default `true`)

## Import

Import is supported using the following syntax:

```shell
terraform import discord_member_roles.example "<server id>:<member id>"
```
