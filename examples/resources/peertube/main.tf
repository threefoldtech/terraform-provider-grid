
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
    name = "peertube"
    cru  = 2
    sru  = 512
    mru  = 4096
  }

  requests {
    name             = "domain"
    public_config    = true
    public_ips_count = 1
  }
}

locals {
  solution_type = "Peertube"
  name          = "peertube"
  node          = grid_scheduler.sched.nodes["peertube"]
}

# this data source is used to break circular dependency in cases similar to the following:
# vm: needs to know the domain in its init script
# gateway_name: needs the ip of the vm to use as backend.
# - the fqdn can be computed from grid_gateway_domain for the vm
# - the backend can reference the vm ip directly
data "grid_gateway_domain" "domain" {
  node = grid_scheduler.sched.nodes["domain"]
  name = local.name
}

resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = [grid_scheduler.sched.nodes["peertube"]]
  ip_range      = "10.1.0.0/16"
  description   = "peertube network"
}
resource "grid_deployment" "d1" {
  node          = grid_scheduler.sched.nodes["peertube"]
  solution_type = local.solution_type
  name          = local.name
  network_name  = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/peertube-v3.1.1.flist"
    cpu        = 2
    entrypoint = "/sbin/zinit init"
    memory     = 4096
    env_vars = {
      SSH_KEY                     = try(length("${var.public_key}"), 0) > 0 ? "${var.public_key}" : file("~/.ssh/id_rsa.pub")
      PEERTUBE_DB_SUFFIX          = "_prod"
      PEERTUBE_DB_USERNAME        = "peertube"
      PEERTUBE_DB_PASSWORD        = "peertube"
      PEERTUBE_ADMIN_EMAIL        = "support@threefold.com"
      PEERTUBE_WEBSERVER_HOSTNAME = data.grid_gateway_domain.domain.fqdn
      PEERTUBE_WEBSERVER_PORT     = 443
      PEERTUBE_SMTP_HOSTNAME      = "https://app.sendgrid.com"
      PEERTUBE_SMTP_USERNAME      = "sendgridusername"
      PEERTUBE_SMTP_PASSWORD      = "sendgridpassword"
      PEERTUBE_BIND_ADDRESS       = "::",
    }
    planetary = true
  }
}

locals {
  ygg_ip = try(length(grid_deployment.d1.vms[0].planetary_ip), 0) > 0 ? grid_deployment.d1.vms[0].planetary_ip : ""
}

resource "grid_name_proxy" "p1" {
  node            = grid_scheduler.sched.nodes["domain"]
  solution_type   = local.solution_type
  name            = local.name
  backends        = [format("http://[%s]:9000", local.ygg_ip)]
  tls_passthrough = false
}

output "fqdn" {
  value = data.grid_gateway_domain.domain.fqdn
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].planetary_ip
}
