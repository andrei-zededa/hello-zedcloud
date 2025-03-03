# The edge-app-instances are created automatically for each edge-node that
# joins the project.

resource "zedcloud_network_instance" "NET_INSTANCES" {
  for_each = {
    for x in lookup(data.restful_resource.EDGE_NODES.output, "list", []) : x.name => x
  }
  name      = "ni_local_nat__${each.value.name}"
  title     = "TF auto-created instance of ni_local_nat for ${each.value.name}"
  kind      = "NETWORK_INSTANCE_KIND_LOCAL"
  type      = "NETWORK_INSTANCE_DHCP_TYPE_V4"
  device_id = each.value.id

  depends_on = [data.restful_resource.EDGE_NODES]

  # TODO: `uplink` is the most common configuration, must double-check
  # why multiple adapters are set as uplink even though no address. Must
  # match the edge-node interface name which is set the same as the
  # "logical label" in the model, could be `eth0` for example.
  port           = "uplink"
  device_default = true

  tags = {
    ni_local_nat = "true"
  }
}

resource "zedcloud_application_instance" "APP_INSTANCES" {
  for_each = {
    for x in lookup(data.restful_resource.EDGE_NODES.output, "list", []) : x.name => x
  }

  depends_on = [
    data.restful_resource.EDGE_NODES,
    zedcloud_network_instance.NET_INSTANCES
  ]

  name      = "${zedcloud_project.PROJECT_1.name}__${zedcloud_application.hello_zedcloud_app_definition.name}__${each.value.name}"
  title     = "TF auto-created instance of ${zedcloud_application.hello_zedcloud_app_definition.name} for ${each.value.name}"
  device_id = each.value.id
  app_id    = zedcloud_application.hello_zedcloud_app_definition.id
  app_type  = zedcloud_application.hello_zedcloud_app_definition.manifest[0].app_type

  activate = true

  logs {
    access = true
  }

  custom_config {
    add                  = true
    allow_storage_resize = false
    override             = false
  }

  manifest_info {
    # bundle_version      = "1"
    transition_action = "INSTANCE_TA_NONE"
  }

  vminfo {
    # required
    cpus = 1
    mode = zedcloud_application.hello_zedcloud_app_definition.manifest[0].vmmode
    vnc  = false
  }

  drives {
    cleartext = true
    mountpath = "/"
    imagename = zedcloud_image.hello_zedcloud_container_image.name
    maxsize   = "0"
    preserve  = false
    readonly  = false
    drvtype   = ""
    target    = ""
  }

  interfaces {
    intfname    = zedcloud_application.hello_zedcloud_app_definition.manifest[0].interfaces[0].name
    intforder   = 1
    privateip   = false
    netinstname = ""
    netinsttag = {
      ni_local_nat = "true"
    }
  }
}

# Track the monitored resources in a local
locals {
  monitored_resource_hash = sha256(jsonencode({
    image_name    = zedcloud_image.hello_zedcloud_container_image.name
    image_title   = zedcloud_image.hello_zedcloud_container_image.title
    image_rel_url = zedcloud_image.hello_zedcloud_container_image.image_rel_url

    app_name  = zedcloud_application.hello_zedcloud_app_definition.name
    app_title = zedcloud_application.hello_zedcloud_app_definition.title
    app_uver  = zedcloud_application.hello_zedcloud_app_definition.user_defined_version
    app_ver   = zedcloud_application.hello_zedcloud_app_definition.manifest[0].ac_version
  }))
}

# Create one null_resource per app instance
resource "null_resource" "APP_INSTANGE_UPGRADE_HOOKS" {
  for_each = zedcloud_application_instance.APP_INSTANCES

  triggers = {
    # Track both the instance ID and the resource hash
    instance_id   = each.key
    resource_hash = local.monitored_resource_hash
  }
}

# Create one restful_operation per app instance
resource "restful_operation" "APP_INSTANGE_UPGRADE_API_CALL" {
  for_each = {
    for k, v in zedcloud_application_instance.APP_INSTANCES :
    # Create a composite key with both the instance ID and the resource hash
    "${k}_${local.monitored_resource_hash}" => {
      id      = k
      inst_id = v.id
    }
  }

  path   = "/apps/instances/id/${each.value.inst_id}/refresh/purge"
  method = "PUT"

  # Static depends_on to the entire null_resource collection
  depends_on = [null_resource.APP_INSTANGE_UPGRADE_HOOKS]
}
