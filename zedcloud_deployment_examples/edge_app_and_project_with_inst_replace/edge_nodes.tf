data "restful_resource" "EDGE_NODES" {
  id = "/devices/status"

  depends_on = [data.zedcloud_project.PROJECT_1]

  query = {
    projectName = [data.zedcloud_project.PROJECT_1.name]
  }
}

output "EDGE_NODES" {
  description = "Print edge-nodes which have joined the project"
  value = {
    for x in lookup(data.restful_resource.EDGE_NODES.output, "list", []) : x.name => {
      id = x.id
    }
  }
}
