
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

resource "random_string" "name" {
  length  = 8
  special = false
}
resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}

provider "grid" {
}

resource "grid_scheduler" "sched" {
  requests {
    name    = "node"
    cru     = 2
    sru     = 1 * 1024
    mru     = 4 * 1024
    farm_id = 1
  }

  requests {
    name             = "gateway"
    public_config    = true
    public_ips_count = 1
  }
}

resource "grid_network" "net1" {
  name          = random_string.name.result
  nodes         = [grid_scheduler.sched.nodes["node"]]
  ip_range      = "10.1.0.0/16"
  description   = "mattermost network"
  add_wg_access = true
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["node"]) = random_bytes.mycelium_key.hex
  }
}

resource "grid_deployment" "d1" {
  name         = random_string.name.result
  node         = grid_scheduler.sched.nodes["node"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/mattermost-latest.flist"
    cpu        = 2
    entrypoint = "/sbin/zinit init"
    memory     = 4096
    env_vars = {
      SSH_KEY      = "${var.public_key}",
      DB_PASSWORD  = "ashroof"
      SITE_URL     = format("https://%s", data.grid_gateway_domain.domain.fqdn)
      SMTPPASSWORD = "password"
      SMTPUSERNAME = "Ashraf"
      SMTPSERVER   = "smtp.gmail.com"
      SMTPPORT     = 587
    }
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
  }
}

locals {
  mycelium_ip = try(length(grid_deployment.d1.vms[0].mycelium_ip), 0) > 0 ? grid_deployment.d1.vms[0].mycelium_ip : ""
}

# this data source is used to break circular dependency in cases similar to the following:
# vm: needs to know the domain in its init script
# gateway_name: needs the ip of the vm to use as backend.
# - the fqdn can be computed from grid_gateway_domain for the vm
# - the backend can reference the vm ip directly 
data "grid_gateway_domain" "domain" {
  node = grid_scheduler.sched.nodes["gateway"]
  name = "testmatter"
}

resource "grid_name_proxy" "p1" {
  name            = "testmatter"
  node            = grid_scheduler.sched.nodes["gateway"]
  backends        = [format("http://[%s]:8000", local.mycelium_ip)]
  tls_passthrough = false
}

output "fqdn" {
  value = data.grid_gateway_domain.domain.fqdn
}

output "mycelium_ip" {
  value = grid_deployment.d1.vms[0].mycelium_ip
}
