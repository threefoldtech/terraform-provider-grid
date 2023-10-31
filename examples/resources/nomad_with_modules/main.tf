module "nomad" {
  source          = "/home/eslam/work/currently/terraform-provider-grid/modules/nomad-module"
  ssh_key         = local.ssh_key
  first_server_ip = local.first_server_ip
  network         = local.network
  servers         = local.servers
  clients         = local.clients
}

locals {
  ssh_key         = file("~/.ssh/id_rsa.pub")
  first_server_ip = "10.1.2.2"

  network = {
    nodes       = [29]
    ip_range    = "10.1.0.0/16"
    name        = "nomadtest"
    description = "new network for nomad"
  }

  servers = [
    {
      name        = "server1"
      node        = 29
      cpu         = 2
      memory      = 1024
      publicip    = false
      publicip6   = false
      planetary   = true
      mount_point = "/mnt"
      disk = {
        name = "server1dsk"
        size = 5
      }
    },
    {
      name        = "server2"
      node        = 29
      cpu         = 2
      memory      = 1024
      publicip    = false
      publicip6   = false
      planetary   = true
      mount_point = "/mnt"
      disk = {
        name = "server2dsk"
        size = 5
      }
    },
    {
      name        = "server3"
      node        = 29
      cpu         = 2
      memory      = 1024
      publicip    = false
      publicip6   = false
      planetary   = true
      mount_point = "/mnt"
      disk = {
        name = "server3dsk"
        size = 5
      }
    },
  ]

  clients = [
    {
      name        = "client1"
      node        = 29
      cpu         = 2
      memory      = 1024
      publicip    = false
      publicip6   = false
      planetary   = true
      mount_point = "/mnt"
      disk = {
        name = "client1dsk"
        size = 5
      }
    },
  ]
}

output "server1_ip" {
  value = module.nomad.server1.vms[0].ygg_ip
}
output "server2_ip" {
  value = module.nomad.servers.vm[0].vms[0].ygg_ip
}
output "server3_ip" {
  value = module.nomad.servers.vm[0].vms[1].ygg_ip
}
output "client1_ip" {
  value = module.nomad.clients.vm[0].vms[0].ygg_ip
}
