module "nomad" {
  source          = "github.com/threefoldtech/terraform-provider-grid/modules/nomad-module"
  ssh_key         = var.ssh_key
  first_server_ip = var.first_server_ip
  network         = var.network
  servers         = var.servers
  clients         = var.clients
}

variable "ssh_key" {
  type = string
}

variable "first_server_ip" {
  type = string
}

variable "network" {
  type = object({
    name        = string
    nodes       = list(number)
    ip_range    = string
    description = string
  })
}

variable "servers" {
  type = list(object({
    name        = string
    node        = number
    cpu         = number
    memory      = number
    rootfs_size = number
    mount_point = string
    disk = object({
      name = string
      size = number
    })
  }))
}

variable "clients" {
  type = list(object({
    name        = string
    node        = number
    cpu         = number
    memory      = number
    rootfs_size = number
    mount_point = string
    disk = object({
      name = string
      size = number
    })
  }))
}

output "server1_ip" {
  value = module.nomad.servers.vm[0].vms[0].ip
}
output "server1_ygg_ip" {
  value = module.nomad.servers.vm[0].vms[0].ygg_ip
}
output "server2_ygg_ip" {
  value = module.nomad.servers.vm[1].vms[0].ygg_ip
}
output "server3_ygg_ip" {
  value = module.nomad.servers.vm[2].vms[0].ygg_ip
}
output "client1_ygg_ip" {
  value = module.nomad.clients.vm[0].vms[0].ygg_ip
}
