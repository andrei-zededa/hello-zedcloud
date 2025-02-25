resource "zedcloud_datastore" "Dockerhub_with_username" {
  name  = "Dockerhub_${var.DOCKERHUB_USERNAME}"
  title = "Dockerhub_${var.DOCKERHUB_USERNAME}"

  ds_type = "DATASTORE_TYPE_CONTAINERREGISTRY"
  ds_fqdn = "docker://docker.io"
  ds_path = var.DOCKERHUB_USERNAME
}
