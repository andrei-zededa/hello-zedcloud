# The variables below can be set either as an environment variable in
# the `TF_VAR_ZEDEDA_CLOUD_URL="zedcloud...."` format, for example, or
# as a `-var="ZEDEDA_CLOUD_URL=zedcloud...."` CLI argument.

# Defined as a secret in the Github repo.
variable "ZEDEDA_CLOUD_URL" {
  description = "ZEDEDA CLOUD URL"
  sensitive   = true
  type        = string
}

# Defined as a secret in the Github repo.
variable "ZEDEDA_CLOUD_TOKEN" {
  description = "ZEDEDA CLOUD API TOKEN"
  sensitive   = true
  type        = string
}

# Defined as a variable in the Github repo.
variable "DOCKERHUB_USERNAME" {
  sensitive = false
  type      = string
  default   = "andreizededa"
}

# Defined as a variable in the Github repo.
variable "DOCKERHUB_IMAGE_NAME" {
  sensitive = false
  type      = string
  default   = "hello-zedcloud"
}

# Most likely this comes from the trigger for the GHA workflow that calls
# terraform. The corresponding `TF_VAR_DOCKERHUB_IMAGE_LATEST_TAG` environment
# variable should be set to override the `latest` default value below.
variable "DOCKERHUB_IMAGE_LATEST_TAG" {
  sensitive = false
  type      = string
  default   = "latest"
}

variable "PROJECT_NAME" {
  sensitive = false
  type      = string
  default   = "hello_zedcloud_example_proj_1"
}
