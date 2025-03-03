# This creates a project in which edge-nodes can be put (joined).
#
# For every edge-node that joins this project a network-instance will be
# created with the simple configuration below (that's basically a Linux
# bridge + an automatically assigned private IPv4 subnet + DHCP server
# + iptables NAT configuration).
#
# Also for every edge-node the app policy will trigger the creation of
# an edge-app-instance that runs the references edge-app and which when
# started will have it's virtual interface connected to the network-instance
# created on the edge-node as above, this is done through matching the
# `ni_local_nat = "true"` tag (which as it can be see is actually a key=value
# pair of strings, not a single freeform string as "tag" would suggest).
resource "zedcloud_project" "PROJECT_1" {
  name        = var.PROJECT_NAME
  title       = var.PROJECT_NAME
  description = "Project ${var.PROJECT_NAME} created via hello-zedcloud/zedcloud_deployment"

  type = "TAG_TYPE_PROJECT"
  tag_level_settings {
    flow_log_transmission = "NETWORK_INSTANCE_FLOW_LOG_TRANSMISSION_UNSPECIFIED"
    # Zedcloud versions starting with 14.1.0 support `interface_ordering` and
    # if it is not specified then it will show up as a diff. Older versions do
    # NOT support it and if specified it will cause an error.
    interface_ordering = "INTERFACE_ORDERING_DISABLED"
  }
}
