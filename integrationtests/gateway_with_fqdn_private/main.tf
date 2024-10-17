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

resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}

resource "random_string" "name" {
  length  = 8
  special = false
}

resource "grid_network" "net1" {
  nodes       = [11]
  ip_range    = "10.1.0.0/16"
  name        = random_string.name.result
  description = "private FQDN gateway network"
  mycelium_keys = {
    format("%s", 11) = random_bytes.mycelium_key.hex
  }
}
resource "grid_deployment" "d1" {
  name         = random_string.name.result
  node         = 11
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
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
  }
}

resource "grid_fqdn_proxy" "p1" {
  node            = 11
  name            = "test"
  fqdn            = var.fqdn
  backends        = [format("http://%s:9000", grid_deployment.d1.vms[0].ip)]
  network         = grid_network.net1.name
  tls_passthrough = false
}

output "fqdn" {
  value = grid_fqdn_proxy.p1.fqdn
}

output "mycelium_ip" {
  value = grid_deployment.d1.vms[0].mycelium_ip
}
