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
  name = "mountDeployment"
}


resource "grid_network" "net1" {
  nodes       = [2, 4]
  ip_range    = "10.1.0.0/16"
  name        = local.name
  description = "newer network"
}
resource "grid_deployment" "d1" {
  name         = local.name
  node         = 2
  network_name = grid_network.net1.name
  disks {
    name        = "data"
    size        = 10
    description = "volume holding app data"
  }
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 1
    publicip   = true
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    mounts {
      disk_name   = "data"
      mount_point = "/app"
    }
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
  }
  vms {
    name       = "anothervm"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 1
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
  }
}
output "wg_config" {
  value = grid_network.net1.access_wg_config
}
output "node1_zmachine1_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "node1_zmachine2_ip" {
  value = grid_deployment.d1.vms[1].ip
}
output "computed_public_ip" {
  value = grid_deployment.d1.vms[0].computedip
}
