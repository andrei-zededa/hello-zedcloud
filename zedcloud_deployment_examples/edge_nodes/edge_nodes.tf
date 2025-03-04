# We define the list of all edge-nodes that will be created as the `devices` array.
locals {
  devices_list = [
    {
      name          = "ENODE_500"
      serial_number = "SN10500"
    },
    {
      name          = "ENODE_501"
      serial_number = "SN10501"
    },
  ]

  # Convert the array to a map with unique keys for easier use with `for_each`.
  devices_map = {
    for index, device in local.devices_list :
    device.name => device
  }
}

variable "onboarding_key" {
  description = "Zedcloud onboarding key"
  type        = string
  default     = "5d0767ee-0547-4569-b530-387e526f8cb9"
}

resource "zedcloud_network" "edge_node_as_dhcp_client" {
  name  = "edge_node_as_dhcp_client_${data.zedcloud_project.PROJECT_1.name}"
  title = "edge_node_as_dhcp_client_${data.zedcloud_project.PROJECT_1.name}"
  kind  = "NETWORK_KIND_V4"

  project_id = data.zedcloud_project.PROJECT_1.id

  ip {
    dhcp = "NETWORK_DHCP_TYPE_CLIENT"
  }
  mtu = 1500
}

resource "zedcloud_edgenode" "EDGE_NODES" {
  for_each = local.devices_map

  name           = each.value.name
  title          = each.value.name
  serialno       = each.value.serial_number
  onboarding_key = var.onboarding_key
  model_id       = zedcloud_model.VM_WITH_MANY_PORTS.id
  project_id     = data.zedcloud_project.PROJECT_1.id
  # The TF provider knows how to do 2 API requests if needed to set a
  # newly created edge node to ADMIN_STATE_ACTIVE.
  admin_state = "ADMIN_STATE_ACTIVE"

  interfaces {
    intfname = "first_physical_intf" # Must match the logical label defined in the model.
    # AdapterUsage Adapter Usage
    #
    # - ADAPTER_USAGE_UNSPECIFIED: Adapter unspecified
    #   - ADAPTER_USAGE_MANAGEMENT: Adapter can be used by EVE as well as other Edge applications
    #   - ADAPTER_USAGE_APP_DIRECT: Adapter is directly used by one edge application
    #   - ADAPTER_USAGE_APP_SHARED: Adapter can be shared by different network instances
    #   - ADAPTER_USAGE_DISABLED: Adapter disabled, for future use
    intf_usage = "ADAPTER_USAGE_MANAGEMENT"
    cost       = 0
    netname    = zedcloud_network.edge_node_as_dhcp_client.name
    tags = {
      # Any string key/value pair should work here.
      net_intf_first = "true"
    }
  }

  interfaces {
    intfname   = "2nd_physical_intf" # Must match the logical label defined in the model.
    intf_usage = "ADAPTER_USAGE_APP_SHARED"
    cost       = 0
    tags = {
      net_intf_second = "true"
    }
  }

  tags = {}
}
