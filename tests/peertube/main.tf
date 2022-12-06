  
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

  # this data source is used to break circular dependency in cases similar to the following:
  # vm: needs to know the domain in its init script
  # gateway_name: needs the ip of the vm to use as backend.
  # - the fqdn can be computed from grid_gateway_domain for the vm
  # - the backend can reference the vm ip directly 
  data "grid_gateway_domain" "domain" {
    node = 3
    name = "ashrafpeertube"
  }
  resource "grid_network" "net1" {
      nodes = [3]
      ip_range = "10.1.0.0/16"
      name = "network"
      description = "newer network"
      add_wg_access = true
  }
  resource "grid_deployment" "d1" {
    node = 2
    network_name = grid_network.net1.name
    vms {
      name = "vm1"
      flist = "https://hub.grid.tf/tf-official-apps/peertube-v3.1.1.flist"
      cpu = 2 
      # publicip = true
      entrypoint = "/sbin/zinit init"
      memory = 4096
      env_vars = {
        TEST_VAR = "this value for test"
        SSH_KEY = "${var.public_key}"
        PEERTUBE_DB_SUFFIX = "_prod"
        PEERTUBE_DB_USERNAME = "peertube"
        PEERTUBE_DB_PASSWORD = "peertube"
        PEERTUBE_ADMIN_EMAIL = "support@threefold.com"
        PEERTUBE_WEBSERVER_HOSTNAME = data.grid_gateway_domain.domain.fqdn
        PEERTUBE_WEBSERVER_PORT = 443
        PEERTUBE_SMTP_HOSTNAME = "https://app.sendgrid.com"
        PEERTUBE_SMTP_USERNAME = "sendgridusername"
        PEERTUBE_SMTP_PASSWORD = "sendgridpassword"
        PEERTUBE_BIND_ADDRESS = "::",
      }
      planetary = true
    }
  }
  resource "grid_name_proxy" "p1" {
    node = 2
    name = "ashrafpeertube"
    backends = [format("http://[%s]:9000", grid_deployment.d1.vms[0].ygg_ip)]
    tls_passthrough = false
  }
  output "fqdn" {
      value = data.grid_gateway_domain.domain.fqdn
  }
  output "node1_zmachine1_ip" {
      value = grid_deployment.d1.vms[0].ip
  }
  output "public_ip" {
      value = split("/",grid_deployment.d1.vms[0].computedip)[0]
  }

  output "ygg_ip" {
      value = grid_deployment.d1.vms[0].ygg_ip
  }

