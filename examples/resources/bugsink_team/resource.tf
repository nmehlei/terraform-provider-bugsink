# NOTE: terraform destroy fails by design on this resource - Bugsink has
# no API to delete a team. See the resource documentation before adopting
# it in any workflow that assumes destroy always succeeds.
resource "bugsink_team" "platform" {
  name       = "platform"
  visibility = "discoverable"
}
