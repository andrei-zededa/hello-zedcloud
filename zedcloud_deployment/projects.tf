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

  app_policy {
    # The name MUST be in the "$PROJECT_NAME.apppolicy" format.
    name  = "${var.PROJECT_NAME}.apppolicy"
    title = ""
    type  = "POLICY_TYPE_APP"

    app_policy {
      apps {
        name              = zedcloud_application.hello_zedcloud_app_definition.name
        title             = ""
        app_id            = zedcloud_application.hello_zedcloud_app_definition.id
        naming_scheme     = "APP_NAMING_SCHEME_PROJECT_APP_DEVICE"
        name_project_part = var.PROJECT_NAME
        name_app_part     = zedcloud_application.hello_zedcloud_app_definition.name

        origin_type = "ORIGIN_UNSPECIFIED"

        cpus     = 0
        memory   = 0
        networks = 0

        manifest_json {
          ac_kind         = ""
          ac_version      = ""
          name            = ""
          app_type        = "APP_TYPE_CONTAINER"
          deployment_type = "DEPLOYMENT_TYPE_STAND_ALONE"
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

        start_delay_in_seconds = 10
      }
    }
  }

  network_policy {
    # The name MUST be in the "$PROJECT_NAME.networkPolicy" format. Very consistent with the app policy naming !
    name  = "${var.PROJECT_NAME}.networkPolicy"
    title = ""
    type  = "POLICY_TYPE_NETWORK"

    network_policy {
      net_instance_config {
        name      = "ni_local_nat"
        title     = "ni_local_nat"
        kind      = "NETWORK_INSTANCE_KIND_LOCAL"
        type      = "NETWORK_INSTANCE_DHCP_TYPE_V4"
        device_id = "" # NOTE: Field is marked as mandatory in the TF provider.

        # TODO: `uplink` is the most common configuration, must doublec-check
        # why multiple adapters are set as uplink even though no address. Must
        # match the edge-node interface name which is set the same as the
        # "logical label" in the model.
        port           = "eth0"
        device_default = true

        tags = {
          ni_local_nat = "true"
        }
      }
    }
  }
}
