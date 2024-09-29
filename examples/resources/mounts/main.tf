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
  name = "mountdeployment"
}


resource "grid_scheduler" "sched" {
  requests {
    name = "node1"
    cru  = 1
    sru  = 1024 * 10
    mru  = 1024
  }
}

resource "grid_network" "net1" {
  nodes       = [grid_scheduler.sched.nodes["node1"]]
  ip_range    = "10.1.0.0/16"
  name        = local.name
  description = "newer network"
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["node1"]) = random_bytes.mycelium_key.hex
  }
}

resource "grid_deployment" "d1" {
  name         = local.name
  node         = grid_scheduler.sched.nodes["node1"]
  network_name = grid_network.net1.name
  disks {
    name        = "data"
    size        = 10
    description = "volume holding app data"
  }
  vms {
    name             = "vm1"
    flist            = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu              = 1
    publicip         = true
    memory           = 1024
    entrypoint       = "/sbin/zinit init"
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
    mounts {
      name        = "data"
      mount_point = "/app"
    }
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
  }
  vms {
    name             = "anothervm"
    flist            = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu              = 1
    memory           = 1024
    entrypoint       = "/sbin/zinit init"
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
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
output "node1_zmachine1_mycelium_ip" {
  value = grid_deployment.d1.vms[0].mycelium_ip
}
output "node1_zmachine2_mycelium_ip" {
  value = grid_deployment.d1.vms[1].mycelium_ip
}
output "computed_public_ip" {
  value = grid_deployment.d1.vms[0].computedip
}
