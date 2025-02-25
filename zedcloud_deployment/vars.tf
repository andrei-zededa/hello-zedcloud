variable "ZEDEDA_CLOUD_URL" {
  description = "ZEDEDA CLOUD URL"
  sensitive   = false
  type        = string
}

variable "ZEDEDA_CLOUD_TOKEN" {
  description = "ZEDEDA CLOUD API TOKEN"
  sensitive   = true
  type        = string
}

variable "DOCKERHUB_USERNAME" {
  sensitive = false
  type      = string
  default   = "andreizededa"
}

variable "DOCKERHUB_IMAGE_NAME" {
  sensitive = false
  type      = string
  default   = "hello-zedcloud"
}

variable "DOCKERHUB_IMAGE_LATEST_TAG" {
  sensitive = false
  type      = string
  default   = "v0.3.9"
}

variable "PROJECT_NAME" {
  sensitive = false
  type      = string
  default   = "hello_zedcloud_example_proj_1"
}

variable "EDGE_NODE_NAME" {
  sensitive = false
  type      = string
  default   = "example_edge_node_1"
}
