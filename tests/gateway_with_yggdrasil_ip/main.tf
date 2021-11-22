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
    nodes = [1]
    ip_range = "10.1.0.0/16"
    name = "network"
    description = "newer network"
}
resource "grid_deployment" "d1" {
  node = 1
  network_name = grid_network.net1.name
  ip_range = grid_network.net1.nodes_ip_range["1"]
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 2 
    publicip = true
    memory = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }
    planetary = true
  }
}

resource "grid_fqdn_proxy" "p1" {
  node = 1
  name = "testname"
  fqdn = "remote.hassan.grid.tf"
  backends = [format("http://[${grid_deployment.d1.vms[0].ygg_ip}]")]
  tls_passthrough = false
}

output "fqdn" {
    value = grid_fqdn_proxy.p1.fqdn
}

output "ygg_ip" {
    value = grid_deployment.d1.vms[0].ygg_ip
}
