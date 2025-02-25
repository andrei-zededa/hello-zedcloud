locals {
  image_version = replace(var.DOCKERHUB_IMAGE_LATEST_TAG, "/^v/", "")
}

resource "zedcloud_application" "hello_zedcloud_app_definition" {
  name  = "${var.DOCKERHUB_IMAGE_NAME}_app_definition"
  title = "${var.DOCKERHUB_IMAGE_NAME}_app_definition"

  networks    = 1
  origin_type = "ORIGIN_LOCAL"

  user_defined_version = local.image_version

  manifest {
    ac_kind             = "PodManifest"
    ac_version          = local.image_version
    app_type            = "APP_TYPE_CONTAINER"
    cpu_pinning_enabled = false
    deployment_type     = "DEPLOYMENT_TYPE_STAND_ALONE"
    enablevnc           = false
    name                = "${var.DOCKERHUB_IMAGE_NAME}_app_definition"
    vmmode              = "HV_PV"

    configuration {
      custom_config {
        add                  = true
        allow_storage_resize = false
        field_delimiter      = "####"
        name                 = "config01"
        override             = false
        # ❯ echo "I2Ns......iMjIyM=" | base64 -d
        # #cloud-config
        # runcmd:
        #   - TEST_VARIABLE_1=http://10.7.0.5:6666,http://10.4.0.5:7777
        #   - TEST_VARIABLE_2=####TEST_VARIABLE_2####
        template = "I2Nsb3VkLWNvbmZpZwpydW5jbWQ6CiAgLSBURVNUX1ZBUklBQkxFXzE9UExBSU5URVhUOi8vMTAuNy4wLjU6NjY2NixQTEFJTlRFWFQ6Ly8xMC40LjAuNTo3Nzc3CiAgLSBURVNUX1ZBUklBQkxFXzI9IyMjI1RFU1RfVkFSSUFCTEVfMiMjIyM="

        variable_groups {
          name     = "Default Group 1"
          required = true

          variables {
            default    = "ABCD1234"
            encode     = "FILE_ENCODING_UNSPECIFIED"
            format     = "VARIABLE_FORMAT_TEXT"
            label      = "Test variable 2 description text goes here"
            max_length = "200"
            name       = "TEST_VARIABLE_2"
            required   = true
          }
        }
      }
    }

    desc {
      agreement_list  = {}
      app_category    = "APP_CATEGORY_UNSPECIFIED"
      category        = "APP_CATEGORY_DEVOPS"
      license_list    = {}
      logo            = {}
      screenshot_list = {}
    }

    images {
      cleartext   = true
      ignorepurge = false
      imageformat = "CONTAINER"
      imageid     = zedcloud_image.hello_zedcloud_container_image.id
      imagename   = zedcloud_image.hello_zedcloud_container_image.name
      maxsize     = "0"
      mountpath   = "/"
      preserve    = false
      readonly    = false
    }

    interfaces {
      directattach = false
      name         = "port_forwarding"
      optional     = false
      privateip    = false

      acls {
        matches {
          type  = "ip"
          value = "0.0.0.0/0"
        }
      }

      acls {
        actions {
          drop       = false
          limit      = false
          limitburst = 0
          limitrate  = 0
          portmap    = true

          portmapto {
            # This is the application instance port.
            app_port = 8080
          }
        }
        matches {
          type  = "protocol"
          value = "tcp"
        }
        matches {
          # This is the edge-node port.
          type  = "lport"
          value = "8080"
        }
        matches {
          # Source address of the traffic.
          type  = "ip"
          value = "0.0.0.0/0"
        }
      }
    }

    owner {
      email   = "support@zededa.com"
      user    = "Zededa Support"
      website = "help.zededa.com"
    }

    resources {
      name  = "resourceType"
      value = "Tiny"
    }
    resources {
      name  = "cpus"
      value = "1"
    }
    resources {
      name  = "memory"
      value = "524288.00"
    }
  }
}
