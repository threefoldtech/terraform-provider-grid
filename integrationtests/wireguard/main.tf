
variable "public_key" {
  type = string
}

terraform {
  required_providers {
    grid = {
      source  = "threefoldtechdev.com/providers/grid"
      version = "0.2"
    }
  }
}

provider "grid" {
  mnemonic = "ring brand park runway cigar month yard venture else wealth surprise attract"
  network = "dev"
}

resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}

resource "grid_scheduler" "scheduler" {
  requests {
    name      = "node"
    cru       = 2
    sru       = 1024
    mru       = 1024
    yggdrasil = true
    wireguard = true
  }
}

resource "grid_network" "net1" {
  nodes         = [grid_scheduler.scheduler.nodes["node"]]
  ip_range      = "10.1.0.0/16"
  name          = "network"
  description   = "wirequard network"
  mycelium_keys = {
    format("%s", grid_scheduler.scheduler.nodes["node"]) = random_bytes.mycelium_key.hex
  }
  add_wg_access = true
}

resource "grid_deployment" "d1" {
  node         = grid_scheduler.scheduler.nodes["node"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-20.04.flist"
    cpu        = 2
    publicip   = false
    memory     = 1024
    entrypoint = "/init.sh"
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
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
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
  }
}

output "wg_config" {
  value = grid_network.net1.access_wg_config
}
output "vm1_wg_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "vm2_wg_ip" {
  value = grid_deployment.d1.vms[1].ip
}

