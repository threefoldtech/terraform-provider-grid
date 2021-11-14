
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
    nodes = [2, 4]
    ip_range = "172.20.0.0/16"
    name = "net1"
    description = "new network"
    add_wg_access = true
}

resource "grid_deployment" "d1" {
  node = 4
  network_name = grid_network.net1.name
  ip_range = grid_network.net1.nodes_ip_range["4"]
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 2 
    memory = 1024
    entrypoint = "/sbin/zinit init"
    env_vars {
      key = "SSH_KEY"
      value = "${var.public_key}"
    }
    env_vars {
      key = "TEST_VAR"
      value = "this value for test"
    }

  }

}

resource "grid_deployment" "d2" {
  node = 2
  network_name = grid_network.net1.name
  ip_range = grid_network.net1.nodes_ip_range["2"]
  vms {
    name = "vm3"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 1
    memory = 1024
    entrypoint = "/sbin/zinit init"
    env_vars {
      key = "SSH_KEY"
      value = "${var.public_key}"
    }

  }
}

output "wg_config" {
    value = grid_network.net1.access_wg_config
}

output "node1_container1_ip" {
    value = grid_deployment.d1.vms[0].ip
}


output "node2_container1_ip" {
    value = grid_deployment.d2.vms[0].ip
}