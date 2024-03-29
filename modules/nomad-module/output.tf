output "servers" {
  value = { for s in grid_deployment.servers : s.name => s... }
}

output "clients" {
  value = { for c in grid_deployment.clients: c.name => c... }
}
