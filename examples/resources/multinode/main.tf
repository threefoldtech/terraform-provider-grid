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
  name = "multinode_deployment"
}


resource "grid_network" "net1" {
  nodes       = [2, 4]
  ip_range    = "172.20.0.0/16"
  name        = local.name
  description = "new network"
}

resource "grid_deployment" "d1" {
  name         = local.name
  node         = 4
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 2
    memory     = 1024
    publicip   = true
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }

  }

}

resource "grid_deployment" "d2" {
  node         = 2
  network_name = grid_network.net1.name
  vms {
    name       = "vm3"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 1
    memory     = 1024
    publicip   = true
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }

  }
}

output "node1_zmachine1_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "node1_zmachine_computed_public_ip" {
  value = grid_deployment.d1.vms[0].computedip
}

output "node2_zmachine1_ip" {
  value = grid_deployment.d2.vms[0].ip
}
output "node2_zmachine_computed_public_ip" {
  value = grid_deployment.d2.vms[0].computedip
}

