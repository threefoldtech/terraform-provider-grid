
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
  key_type = "ed25519"
}

resource "grid_network" "net1" {
  nodes         = [3]
  ip_range      = "10.1.0.0/16"
  name          = "network"
  description   = "newer network"
  add_wg_access = false
}

resource "grid_deployment" "d1" {
  node         = 3
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-20.04.flist"
    cpu        = 2
    publicip   = false
    memory     = 1024
    entrypoint = "/init.sh"
    env_vars = {
      SSH_KEY  = "${var.public_key}"
    }
    planetary = true
  }
  vms {
    name       = "anothervm"
    flist      = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-20.04.flist"
    cpu        = 1
    memory     = 1024
    entrypoint = "/init.sh"
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }
  }
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
