
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

resource "grid_scheduler" "scheduler" {
  requests {
    name = "node1"
    cru  = 1
    sru  = var.disk_size * 1024
    mru  = 1024
  }
}

resource "grid_network" "net1" {
  nodes       = [grid_scheduler.scheduler.nodes["node1"]]
  ip_range    = "10.1.0.0/16"
  name        = "network"
  description = "newer network"
}
resource "grid_deployment" "d1" {
  node         = grid_scheduler.scheduler.nodes["node1"]
  network_name = grid_network.net1.name
  disks {
    name        = "data"
    size        = var.disk_size
    description = "volume holding app data"
  }
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 1
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    mounts {
      disk_name   = "data"
      mount_point = "/app"
    }
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }
    planetary = true

  }
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].ygg_ip
}