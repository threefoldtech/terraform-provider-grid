
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
    nodes = [1]
    ip_range = "10.1.0.0/16"
    name = "network"
    description = "newer network"
    add_wg_access = true
}
resource "grid_deployment" "d1" {
  node = 2
  network_name = grid_network.net1.name
  ip_range = lookup(grid_network.net1.nodes_ip_range, 2, "")
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 2 
    publicip = true
    memory = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "${var.public_key}"
      TEST_VAR = "this value for test"

    }
    planetary = true
  }
  vms {
    name = "anothervm"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 2
    memory = 1024
    publicip = true
    entrypoint = "/sbin/zinit init"
    
    env_vars = {
      SSH_KEY = "${var.public_key}"
      TEST_VAR = "this value for test"

    
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
output "public_ip" {
    value = grid_deployment.d1.vms[0].computedip
}
output "public_ip" {
    value = grid_deployment.d1.vms[1].computedip
}
output "ygg_ip" {
    value = grid_deployment.d1.vms[0].ygg_ip
}