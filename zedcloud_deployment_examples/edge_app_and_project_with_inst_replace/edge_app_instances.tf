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

  lifecycle {
    replace_triggered_by = [
      zedcloud_image.hello_zedcloud_container_image.name,
      zedcloud_image.hello_zedcloud_container_image.title,
      zedcloud_image.hello_zedcloud_container_image.image_rel_url,
      zedcloud_application.hello_zedcloud_app_definition.name,
      zedcloud_application.hello_zedcloud_app_definition.title,
      zedcloud_application.hello_zedcloud_app_definition.user_defined_version,
      zedcloud_application.hello_zedcloud_app_definition.manifest[0].ac_version,
    ]
  }

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
    transition_action = "INSTANCE_TA_NONE"
  }

  vminfo {
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

output "EDGE_APP_INSTANCES" {
  description = "Print edge-app-instances which have been created for every edge-node which joined the project"
  value = {
    for x in zedcloud_application_instance.APP_INSTANCES : x.name => {
      id = x.id
    }
  }
}
