data "restful_resource" "EDGE_NODES" {
  id = "/devices/status"

  depends_on = [zedcloud_project.PROJECT_1]

  query = {
    projectName = [zedcloud_project.PROJECT_1.name]
  }
}

output "EDGE_NODES" {
  description = "Edge-nodes which have joined the project"
  value = {
    for x in lookup(data.restful_resource.EDGE_NODES.output, "list", []) : x.name => {
      id = x.id
    }
  }
}
