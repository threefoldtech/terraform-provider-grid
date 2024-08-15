
variable "public_key" {
  type = string
}

variable "disk_size" {
  type = number
}
variable "mount_point" {
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

resource "grid_scheduler" "scheduler" {
  requests {
    name = "node"
    cru  = 1
    sru  = var.disk_size * 1024 + 1024
    mru  = 1024
  }
}

resource "grid_network" "net1" {
  nodes       = [grid_scheduler.scheduler.nodes["node"]]
  ip_range    = "10.1.0.0/16"
  name        = random_string.name.result
  description = "vm network"
}

resource "grid_deployment" "d1" {
  node         = grid_scheduler.scheduler.nodes["node"]
  network_name = grid_network.net1.name
  disks {
    name        = "data"
    size        = var.disk_size
    description = "volume holding app data"
  }
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-20.04.flist"
    cpu        = 1
    memory     = 1024
    entrypoint = "/init.sh"
    mounts {
      disk_name   = "data"
      mount_point = "/${var.mount_point}"
    }
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }
    planetary = true
  }
}
output "vm_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "ygg_ip" {
  value = grid_deployment.d1.vms[0].planetary_ip
}
