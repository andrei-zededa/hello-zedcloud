resource "zedcloud_image" "hello_zedcloud_container_image" {
  name  = "${var.DOCKERHUB_IMAGE_NAME}_container_image"
  title = "${var.DOCKERHUB_IMAGE_NAME}_container_image"

  datastore_id = zedcloud_datastore.Dockerhub_with_username.id
  datastore_id_list = [
    zedcloud_datastore.Dockerhub_with_username.id
  ]

  # The final URL that an edge-node will use to download this image will be
  #     ${ds_fqdn}/${ds_path}/${image_rel_url}
  #
  # See also the comment for the datastore.
  image_rel_url    = "${var.DOCKERHUB_IMAGE_NAME}:${var.DOCKERHUB_IMAGE_LATEST_TAG}"
  image_format     = "CONTAINER"
  image_arch       = "AMD64"
  image_type       = "IMAGE_TYPE_APPLICATION"
  image_size_bytes = 0
}
