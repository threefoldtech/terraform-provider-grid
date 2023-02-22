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
  name = "myvm"
}
# this data source is used to break circular dependency in cases similar to the following:
# vm: needs to know the domain in its init script
# gateway_name: needs the ip of the vm to use as backend.
# - the fqdn can be computed from grid_gateway_domain for the vm
# - the backend can reference the vm ip directly 
data "grid_gateway_domain" "domain" {
  node = 7
  name = "ashraf"
}

resource "grid_network" "net1" {
  nodes         = [8]
  ip_range      = "10.1.0.0/24"
  name          = local.name
  description   = "newer network"
  add_wg_access = true
}
resource "grid_deployment" "d1" {
  name         = local.name
  node         = 8
  network_name = grid_network.net1.name
  vms {
    name     = "vm1"
    flist    = "https://hub.grid.tf/tf-official-apps/strm-helloworld-http-latest.flist"
    cpu      = 2
    publicip = true
    memory   = 1024
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
    planetary = true
  }
}
resource "grid_name_proxy" "p1" {
  node            = 7
  name            = "ashraf"
  backends        = [format("http://%s", split("/", grid_deployment.d1.vms[0].computedip)[0])]
  tls_passthrough = false
}
output "fqdn" {
  value = data.grid_gateway_domain.domain.fqdn
}
output "node1_zmachine1_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "computed_public_ip" {
  value = split("/", grid_deployment.d1.vms[0].computedip)[0]
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].ygg_ip
}

