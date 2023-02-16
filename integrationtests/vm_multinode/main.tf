
variable "public_key" {
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
  nodes       = [33, 34]
  ip_range    = "172.20.0.0/16"
  name        = "net1"
  description = "new network"
}

resource "grid_deployment" "d1" {
  node        = 33
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
  node        = 34
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
