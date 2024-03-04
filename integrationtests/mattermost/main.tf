
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
    name = "mattermost_instance"
    cru  = 2
    sru  = 512
    mru  = 4096
  }

  requests {
    name             = "gateway"
    public_config    = true
    public_ips_count = 1
  }
}

# this data source is used to break circular dependency in cases similar to the following:
# vm: needs to know the domain in its init script
# gateway_name: needs the ip of the vm to use as backend.
# - the fqdn can be computed from grid_gateway_domain for the vm
# - the backend can reference the vm ip directly 
data "grid_gateway_domain" "domain" {
  node = grid_scheduler.sched.nodes["gateway"]
  name = "khaledmatter"
}
resource "grid_network" "net1" {
  nodes         = [grid_scheduler.sched.nodes["mattermost_instance"]]
  ip_range      = "10.1.0.0/16"
  name          = "networkk"
  description   = "newer network"
  add_wg_access = true
}
resource "grid_deployment" "d1" {
  node         = grid_scheduler.sched.nodes["mattermost_instance"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/mattermost-latest.flist"
    cpu        = 2
    entrypoint = "/sbin/zinit init"
    memory     = 4096
    env_vars = {
      SSH_KEY      = "${var.public_key}",
      DB_PASSWORD  = "khaled"
      SITE_URL     = format("https://%s", data.grid_gateway_domain.domain.fqdn)
      SMTPPASSWORD = "password"
      SMTPUSERNAME = "Ashraf"
      SMTPSERVER   = "smtp.gmail.com"
      SMTPPORT     = 587
    }
    planetary = true
  }
}

locals {
  ygg_ip = try(length(grid_deployment.d1.vms[0].planetary_ip), 0) > 0 ? grid_deployment.d1.vms[0].planetary_ip : ""
}

resource "grid_name_proxy" "p1" {
  node            = grid_scheduler.sched.nodes["gateway"]
  name            = "khaledmatter"
  backends        = [format("http://[%s]:8000", local.ygg_ip)]
  tls_passthrough = false
}
output "fqdn" {
  value = data.grid_gateway_domain.domain.fqdn
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].planetary_ip
}

