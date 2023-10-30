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
    nodes       = [29, 27]
    ip_range    = "10.1.0.0/16"
    name        = "nomadtest"
    description = "new network for nomad"
  }

  servers = [
    {
      name      = "server1"
      node      = 29
      cpu       = 2
      memory    = 1024
      planetary = false
    },
    {
      name      = "server2"
      node      = 27
      cpu       = 2
      memory    = 1024
      planetary = false
    },
    {
      name      = "server3"
      node      = 27
      cpu       = 2
      memory    = 1024
      planetary = false
    },
  ]

  clients = [
    {
      name      = "client1"
      node      = 27
      cpu       = 2
      memory    = 1024
      planetary = false
    },
  ]
}

output "servers" {
  value = module.nomad.servers
}

output "clients" {
  value = module.nomad.clients
}
