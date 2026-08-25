# NOTE: terraform destroy fails by design on this resource - Bugsink has
# no API to delete a project. See the resource documentation before
# adopting it in any workflow that assumes destroy always succeeds.
#
# Changing `team` forces replacement: Bugsink has no API to move a project
# between teams.
resource "bugsink_project" "checkout" {
  team                      = bugsink_team.platform.id
  name                      = "checkout-service"
  alert_on_new_issue        = true
  alert_on_regression       = true
  retention_max_event_count = 10000
}

# dsn is computed and sensitive - only available after the resource is
# created, since Bugsink's create response doesn't include it.
output "checkout_dsn" {
  value     = bugsink_project.checkout.dsn
  sensitive = true
}
