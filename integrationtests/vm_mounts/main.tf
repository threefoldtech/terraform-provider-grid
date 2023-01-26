
variable "public_key" {
  type = string
}

variable "disk_size" {
  type = number
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
  nodes         = [2]
  ip_range      = "10.1.0.0/16"
  name          = "network"
  description   = "newer network"
}
resource "grid_deployment" "d1" {
  node         = 2
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
    publicip   = false
    memory     = 1024
    entrypoint = "/init.sh"
    mounts {
      disk_name   = "data"
      mount_point = "/app"
    }
    env_vars = {
      SSH_KEY  = "${var.public_key}"
    }
      planetary = true
  }
}
output "wg_config" {
  value = grid_network.net1.access_wg_config
}
output "node1_container1_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "node1_container2_ip" {
  value = grid_deployment.d1.vms[1].ip
}
output "ygg_ip" {
  value = grid_deployment.d1.vms[0].ygg_ip
}
