
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
    nodes = [2]
    ip_range = "10.1.0.0/16"
    name = "network"
    description = "newer network"
    add_wg_access = true
}
resource "grid_deployment" "d1" {
  node = 2
  network_name = grid_network.net1.name
  disks {
    name = "data"
    size = 10
    description = "volume holding app data"
  }
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-20.04.flist"
    cpu = 1
    publicip = true
    memory = 1024
    entrypoint = "/init.sh"
    mounts {
        disk_name = "data"
        mount_point = "/app"
    }
    env_vars = {
      SSH_KEY = "${var.public_key}"
      TEST_VAR = "this value for test"
    }
  }
  vms {
    name = "anothervm"
    flist = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-20.04.flist"
    cpu = 1
    memory = 1024
    entrypoint = "/init.sh"
    env_vars = {
      SSH_KEY = "${var.public_key}"
      TEST_VAR = "this value for test"
    }
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
output "public_ip" {
    value = grid_deployment.d1.vms[0].computedip
}
