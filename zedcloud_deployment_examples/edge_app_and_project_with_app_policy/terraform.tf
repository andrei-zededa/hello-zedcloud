terraform {
  required_providers {
    zedcloud = {
      # Check for the latest public releases of the Zededa zedcloud terraform
      # provider here: https://registry.terraform.io/providers/zededa/zedcloud/latest/docs
      # or here: https://search.opentofu.org/provider/zededa/zedcloud/latest .
      #
      # NOTE: If `.terraform.lock.hcl` is commited to the git repository then
      # it must be updated for changes to the provider version.
      #
      # source = "zededa/zedcloud" 
      # version = "2.3.0"

      # TODO: This is needed only temporary, remove once a new release of
      # the provider is available from `zededa/terraform-provider/zedcloud`.
      # See also the comment in the corresponding GHA workflow.
      source  = "localhost/zededa/zedcloud"
      version = "0.0.0-dev.fiximageupdate.1"
    }
  }
}

provider "zedcloud" {
  zedcloud_url   = var.ZEDEDA_CLOUD_URL
  zedcloud_token = var.ZEDEDA_CLOUD_TOKEN
}
