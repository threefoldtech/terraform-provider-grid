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

resource "random_string" "name" {
  length  = 8
  special = false
}

locals {
  vm_disk_size = 2
  vm_memory    = 2048
}

provider "grid" {
}
resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}
resource "grid_scheduler" "scheduler" {
  requests {
    name = "node"
    cru  = 0
    sru  = local.vm_disk_size * 1024
    mru  = local.vm_memory
  }
}

resource "grid_network" "net1" {
  nodes       = [grid_scheduler.scheduler.nodes["node"]]
  ip_range    = "10.1.0.0/16"
  name        = random_string.name.result
  description = "vm network"
  mycelium_keys = {
    format("%s",grid_scheduler.scheduler.nodes["node"]) = random_bytes.mycelium_key.hex
  }

}
resource "grid_deployment" "d1" {
  node         = grid_scheduler.scheduler.nodes["node"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 0
    memory     = local.vm_memory
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
    planetary = true
  }
}

