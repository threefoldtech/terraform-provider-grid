
variable "public_key" {
  type = string
}

terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
      version = "0.1.1"
    }
  }
}

provider "grid" {
}

resource "grid_network" "net1" {
    nodes = [2, 4]
    ip_range = "10.1.0.0/16"
    name = "network"
    description = "newer network"
}
resource "grid_deployment" "d1" {
  node = 4
  network_name = grid_network.net1.name
  ip_range = grid_network.net1.nodes_ip_range[2]
  disks {
    name = "data"
    size = 1
    description = "volume holding app data"
  }
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 1
    publicip = true
    memory = 2097152
    entrypoint = "/sbin/zinit init"
    mounts {
        disk_name = "data"
        mount_point = "/app"
    }
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
output "wg_config" {
    value = grid_network.net1.access_wg_config
}
output "node1_container1_ip" {
    value = grid_deployment.d1.vms[0].ip
}

output "public_ip" {
    value = grid_deployment.d1.vms[0].computedip
}
