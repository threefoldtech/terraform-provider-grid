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

locals {
  name = "luihkybveruvytc"
}


resource "grid_network" "net1" {
  nodes         = [34]
  ip_range      = "10.1.0.0/16"
  name          = local.name
  description   = "newer network"
}
resource "grid_deployment" "d1" {
  name         = local.name
  node         = 34
  network_name = grid_network.net1.name
  vms {
    name  = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu   = 2
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }
    planetary = true
  }
}

resource "grid_fqdn_proxy" "p1" {
  node = 15
  name = "test"
  fqdn = "${var.fqdn}"
  backends = [format("http://[%s]:9000", grid_deployment.d1.vms[0].ygg_ip)]
  tls_passthrough = false
}

output "fqdn" {
    value = grid_fqdn_proxy.p1.fqdn
}

output "ygg_ip"{
  value = grid_deployment.d1.vms[0].ygg_ip
}