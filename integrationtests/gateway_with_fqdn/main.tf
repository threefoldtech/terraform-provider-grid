variable "fqdn" {
  type = string
}

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

resource "grid_scheduler" "sched" {
  requests {
    name = "node1"
    cru  = 2
    sru  = 512
    mru  = 1024
  }
  requests {
    name = "gateway"
    public_config   = true
  }
}

locals {
  name = "luihkybveruvytc"
}


resource "grid_network" "net1" {
  nodes       = [grid_scheduler.sched.nodes["node1"]]
  ip_range    = "10.1.0.0/16"
  name        = local.name
  description = "newer network"
}
resource "grid_deployment" "d1" {
  name         = local.name
  node         = grid_scheduler.sched.nodes["node1"]
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
  ygg_ip = try(length(grid_deployment.d1.vms[0].ygg_ip), 0) > 0 ? grid_deployment.d1.vms[0].ygg_ip : ""
}

resource "grid_fqdn_proxy" "p1" {
  node            = grid_scheduler.sched.nodes["gateway"]
  name            = "test"
  fqdn            = var.fqdn
  backends        = [format("http://[%s]:9000", local.ygg_ip)]
  tls_passthrough = false
}

output "fqdn" {
  value = grid_fqdn_proxy.p1.fqdn
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].ygg_ip
}