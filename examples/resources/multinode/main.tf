terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "random_bytes" "d1_mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "d2_mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "d1_mycelium_key" {
  length = 32
}

resource "random_bytes" "d2_mycelium_key" {
  length = 32
}

locals {
  name = "multinodedeployment"
}

resource "grid_scheduler" "sched" {
  requests {
    name             = "node1"
    cru              = 1
    sru              = 1024
    mru              = 1024
    public_ips_count = 1
    public_config    = true
  }
  requests {
    name             = "node2"
    cru              = 1
    sru              = 1024
    mru              = 1024
    public_ips_count = 1
    public_config    = true
  }
}

resource "grid_network" "net1" {
  nodes       = [grid_scheduler.sched.nodes["node1"], grid_scheduler.sched.nodes["node2"]]
  ip_range    = "172.20.0.0/16"
  name        = local.name
  description = "new network"
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["node1"]) = random_bytes.d1_mycelium_key.hex
    format("%s", grid_scheduler.sched.nodes["node2"]) = random_bytes.d2_mycelium_key.hex
  }
}

resource "grid_deployment" "d1" {
  name         = local.name
  node         = grid_scheduler.sched.nodes["node1"]
  network_name = grid_network.net1.name
  vms {
    name             = "vm1"
    flist            = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu              = 2
    memory           = 1024
    publicip         = true
    entrypoint       = "/sbin/zinit init"
    mycelium_ip_seed = random_bytes.d1_mycelium_ip_seed.hex
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
  }
}

resource "grid_deployment" "d2" {
  node         = grid_scheduler.sched.nodes["node2"]
  network_name = grid_network.net1.name
  vms {
    name             = "vm2"
    flist            = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu              = 1
    memory           = 1024
    publicip         = true
    entrypoint       = "/sbin/zinit init"
    mycelium_ip_seed = random_bytes.d2_mycelium_ip_seed.hex
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
  }
}

output "node1_zmachine1_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "node1_zmachine_computed_public_ip" {
  value = grid_deployment.d1.vms[0].computedip
}
output "node1_zmachine_mycelium_ip" {
  value = grid_deployment.d1.vms[0].mycelium_ip
}

output "node2_zmachine1_ip" {
  value = grid_deployment.d2.vms[0].ip
}
output "node2_zmachine_computed_public_ip" {
  value = grid_deployment.d2.vms[0].computedip
}
output "node2_zmachine_mycelium_ip" {
  value = grid_deployment.d2.vms[0].mycelium_ip
}

