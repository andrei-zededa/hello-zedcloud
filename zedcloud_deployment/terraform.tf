terraform {
  required_providers {
    zedcloud = {
      # Public terraform provider release: https://registry.terraform.io/providers/zededa/zedcloud/latest/docs
      source  = "localhost/zededa/zedcloud"
      version = "0.0.0-dev.fiximageupdate.1"
    }
  }
}

provider "zedcloud" {
  zedcloud_url   = var.ZEDEDA_CLOUD_URL
  zedcloud_token = var.ZEDEDA_CLOUD_TOKEN
}
