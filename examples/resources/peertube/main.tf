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

# this data source is used to break circular dependency in cases similar to the following:
# vm: needs to know the domain in its init script
# gateway_name: needs the ip of the vm to use as backend.
# - the fqdn can be computed from grid_gateway_domain for the vm
# - the backend can reference the vm ip directly  
locals {
  solution_type = "Peertube"
  name          = "ashraftube"
  node          = 34
}
data "grid_gateway_domain" "domain" {
  node = 45
  name = local.name
}

resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = [local.node]
  ip_range      = "10.1.0.0/16"
  description   = "newer network"
}
resource "grid_deployment" "d1" {
  node          = local.node
  solution_type = local.solution_type
  name          = local.name
  network_name  = grid_network.net1.name
  ip_range      = lookup(grid_network.net1.nodes_ip_range, local.node, "")
  vms {
    name  = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/peertube-v3.1.1.flist"
    cpu   = 2
    # publicip = true
    entrypoint = "/sbin/zinit init"
    memory     = 4096
    env_vars = {
      SSH_KEY                     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP ashraf@thinkpad"
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
resource "grid_name_proxy" "p1" {
  node            = 45
  solution_type   = local.solution_type
  name            = local.name
  backends        = [format("http://[%s]:9000", grid_deployment.d1.vms[0].ygg_ip)]
  tls_passthrough = false
}
output "fqdn" {
  value = data.grid_gateway_domain.domain.fqdn
}
output "node1_zmachine1_ip" {
  value = grid_deployment.d1.vms[0].ip
}
# output "public_ip" {
#     value = split("/",grid_deployment.d1.vms[0].computedip)[0]
# }

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].ygg_ip
}

