
variable "public_key" {
  type = string
}

terraform {
  required_providers {
    grid = {
      source  = "threefoldtechdev.com/providers/grid"
      version = "0.2"
    }
  }
}

provider "grid" {
}

resource "grid_scheduler" "scheduler" {
  requests {
    name = "node1"
    cru  = 2
    sru  = 512
    mru  = 1024
  }

  requests {
    name = "node2"
    cru  = 1
    sru  = 512
    mru  = 1024
  }
}

resource "grid_network" "net1" {
  nodes = distinct([
    grid_scheduler.scheduler.nodes["node1"],
    grid_scheduler.scheduler.nodes["node2"]
  ])
  ip_range    = "172.20.0.0/16"
  name        = "net1"
  description = "new network"
}

resource "grid_deployment" "d1" {
  node         = grid_scheduler.scheduler.nodes["node1"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-20.04.flist"
    cpu        = 2
    memory     = 1024
    entrypoint = "/init.sh"
    env_vars = {
      SSH_KEY = "${var.public_key}"
      machine = "machine1"
    }
    planetary = true
  }

}

resource "grid_deployment" "d2" {
  node         = grid_scheduler.scheduler.nodes["node2"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm2"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 1
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "${var.public_key}"
      machine = "machine2"
    }
    planetary = true

  }
}

output "vm1_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "vm1_ygg_ip" {
  value = grid_deployment.d1.vms[0].ygg_ip
}

output "vm2_ip" {
  value = grid_deployment.d2.vms[0].ip
}
output "vm2_ygg_ip" {
  value = grid_deployment.d2.vms[0].ygg_ip
}
