output "nomad" {
  value = { for c in grid_deployment.nomad.vms : c.name => c }
}
