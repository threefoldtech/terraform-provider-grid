
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
    nodes = [2, 3]
    ip_range = "172.20.0.0/16"
    name = "net1"
    description = "new network"
}

resource "grid_deployment" "d1" {
  node = 2
  network_name = grid_network.net1.name
  ip_range = lookup(grid_network.net1.nodes_ip_range, 2, "")
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 2 
    memory = 1024
    publicip = true
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }

  }

}

resource "grid_deployment" "d2" {
  node = 3
  network_name = grid_network.net1.name
  ip_range = lookup(grid_network.net1.nodes_ip_range, 3, "")
  vms {
    name = "vm3"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 1
    memory = 1024
    publicip = true
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }

  }
}

output "node1_zmachine1_ip" {
    value = grid_deployment.d1.vms[0].ip
}
output "node1_zmachine_public_ip" {
   value = grid_deployment.d1.vms[0].computedip
}

output "node2_zmachine1_ip" {
    value = grid_deployment.d2.vms[0].ip
}
output "node2_zmachine_public_ip" {
   value = grid_deployment.d2.vms[0].computedip
}

