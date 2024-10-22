variable "fqdn" {
  type = string
}

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
}

resource "grid_scheduler" "sched" {
  requests {
    name      = "node"
    cru       = 2
    sru       = 512
    mru       = 1024
    farm_id   = 1
    yggdrasil = true
    wireguard = false
  }
}

resource "random_string" "name" {
  length  = 8
  special = false
}

resource "grid_network" "net1" {
  name        = random_string.name.result
  nodes       = [grid_scheduler.sched.nodes["node"]]
  ip_range    = "10.1.0.0/16"
  description = "fqdn network"
}
resource "grid_deployment" "d1" {
  name         = random_string.name.result
  node         = grid_scheduler.sched.nodes["node"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 2
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }
    planetary = true
  }
}


locals {
  ygg_ip = try(length(grid_deployment.d1.vms[0].planetary_ip), 0) > 0 ? grid_deployment.d1.vms[0].planetary_ip : ""
}

resource "grid_fqdn_proxy" "p1" {
  node            = 11
  name            = "test"
  fqdn            = var.fqdn
  backends        = [format("http://[%s]:9000", local.ygg_ip)]
  tls_passthrough = false
}

output "fqdn" {
  value = grid_fqdn_proxy.p1.fqdn
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].planetary_ip
}
