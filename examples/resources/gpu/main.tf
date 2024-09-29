terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}
provider "grid" {
}

resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}

locals {
  name    = "testvm"
  node_id = 93
}

resource "grid_network" "net1" {
  name        = local.name
  nodes       = [local.node_id]
  ip_range    = "10.1.0.0/16"
  description = "newer network"
  mycelium_keys = {
    format("%s", local.node_id) = random_bytes.mycelium_key.hex
  }
}
resource "grid_deployment" "d1" {
  name         = local.name
  node         = local.node_id
  network_name = grid_network.net1.name
  disks {
    name        = "data"
    size        = 100
    description = "volume holding app data"
  }
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-vms/ubuntu-22.04.flist"
    cpu        = 4
    memory     = 1024 * 4
    entrypoint = "/init.sh"
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
    mounts {
      name        = "data"
      mount_point = "/app"
    }
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
    gpus = [
      "0000:0e:00.0/1002/744c" //choose the correct gpu id available on the node your are deploying on
    ]
  }


}
output "vm1_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "vm1_mycelium_ip" {
  value = grid_deployment.d1.vms[0].mycelium_ip
}

