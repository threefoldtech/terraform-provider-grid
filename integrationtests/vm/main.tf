
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

resource "random_string" "name" {
  length  = 8
  special = false
}

locals {
  vm_disk_size = 2
  vm_memory    = 2048
}

resource "grid_scheduler" "sched" {
  requests {
    name      = "node"
    cru       = 2
    sru       = local.vm_disk_size * 1024
    mru       = local.vm_memory
    farm_id   = 1
    yggdrasil = true
    wireguard = false
  }
}

resource "grid_network" "net1" {
  nodes       = [grid_scheduler.sched.nodes["node"]]
  ip_range    = "10.1.0.0/16"
  name        = random_string.name.result
  description = "vm network"
}

resource "grid_deployment" "d1" {
  node         = grid_scheduler.sched.nodes["node"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-20.04.flist"
    cpu        = 2
    memory     = local.vm_memory
    entrypoint = "/init.sh"
    env_vars = {
      SSH_KEY  = "${var.public_key}"
      TEST_VAR = "this value for test"
    }
    planetary = true
  }
}
output "ygg_ip" {
  value = grid_deployment.d1.vms[0].planetary_ip
}

output "vm_ip" {
  value = grid_deployment.d1.vms[0].ip
}
