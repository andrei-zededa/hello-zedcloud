# The project is not managed by this terraform configuration, we just need to
# retrieve it's details from Zedcloud.
data "zedcloud_project" "PROJECT_1" {
  name  = var.PROJECT_NAME
  title = var.PROJECT_NAME
  type = "TAG_TYPE_PROJECT"
}
