terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}
provider "grid" {
}

locals {
  name  = "myvm"
  name2 = "myvm2"
  node  = 34
  node2 = 49
}

resource "grid_network" "net1" {
  name        = local.name
  nodes       = [local.node, local.node2]
  ip_range    = "10.1.0.0/16"
  description = "newer network"
}
resource "grid_deployment" "d1" {
  name         = local.name
  node         = local.node
  network_name = grid_network.net1.name
  vms {
    name  = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu   = 2
    # publicip   = true
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
  }
}

resource "grid_deployment" "d2" {
  name         = local.name2
  node         = local.node2
  network_name = grid_network.net1.name
  vms {
    name       = "vm2"
    flist      = "https://hub.grid.tf/azmy.3bot/forwarder.flist"
    cpu        = 2
    publicip   = true
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
      TARGET  = grid_deployment.d1.vms[0].ip
    }
  }
}

output "computed_public_ip" {
  value = grid_deployment.d2.vms[0].computedip
}
