
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
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "grid_network" "net1" {
  nodes       = [33]
  ip_range    = "10.1.0.0/16"
  name        = "network"
  description = "newer network"
}
resource "grid_deployment" "d1" {
  node         = 33
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
  value = grid_deployment.d1.vms[0].ygg_ip
}
