output "master" {
  value = grid_deployment.master.vms[0]
}

output "workers" {
  value = { for w in local.vms_list : w.name => w }
}
