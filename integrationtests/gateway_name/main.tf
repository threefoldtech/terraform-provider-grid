variable "public_key" {
  type = string
}

terraform {
  required_providers {
    grid = {
      source = "threefoldtechdev.com/providers/grid"
    }
  }
}

provider "grid" {
}

# this data source is used to break circular dependency in cases similar to the following:
# vm: needs to know the domain in its init script
# gateway_name: needs the ip of the vm to use as backend.
# - the fqdn can be computed from grid_gateway_domain for the vm
# - the backend can reference the vm ip directly 
data "grid_gateway_domain" "domain" {
  node_id= 14 
  name = "examp123456"
}

locals {
  name = "vmtesting"
}


resource "grid_network" "net1" {
  nodes         = [34]
  ip_range      = "10.1.0.0/16"
  name          = local.name
  description   = "newer network"
}
resource "grid_deployment" "d1" {
  name         = local.name
  node_id        = 34
  network_name = grid_network.net1.name
  vms {
    name  = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu   = 2
    # publicip   = true
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }
    planetary = true
  }
}

resource "grid_name_proxy" "p1" {
  node_id= 14
  name = "examp123456"
  backends = [format("http://[%s]:9000", grid_deployment.d1.vms[0].ygg_ip)]
  tls_passthrough = false
}
output "fqdn" {
    value = data.grid_gateway_domain.domain.fqdn
}

output "ygg_ip"{
  value = grid_deployment.d1.vms[0].ygg_ip
}