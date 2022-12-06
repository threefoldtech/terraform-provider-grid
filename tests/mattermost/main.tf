 
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
    node = 2
    name = "khaledmatter"
  }
  resource "grid_network" "net1" {
      nodes = [2]
      ip_range = "10.1.0.0/16"
      name = "networkk"
      description = "newer network"
      add_wg_access = true
  }
  resource "grid_deployment" "d1" {
    node = 2
    network_name = grid_network.net1.name
    vms {
      name = "vm1"
      flist = "https://hub.grid.tf/tf-official-apps/mattermost-latest.flist"
      cpu = 2 
      entrypoint = "/sbin/zinit init"
      memory = 4096
      env_vars = {
        TEST_VAR = "this value for test"
        SSH_KEY = "${var.public_key}",
        DB_PASSWORD = "khaled"
        SITE_URL = format("https://%s", data.grid_gateway_domain.domain.fqdn)
        SMTPPASSWORD = "password"
        SMTPUSERNAME="Ashraf"
        SMTPSERVER="smtp.gmail.com"
        SMTPPORT = 587
      }
      planetary = true
    }
  }
  resource "grid_name_proxy" "p1" {
    node = 2
    name = "khaledmatter"
    backends = [format("http://[%s]:8000", grid_deployment.d1.vms[0].ygg_ip)]
    tls_passthrough = false
  }
  output "fqdn" {
      value = data.grid_gateway_domain.domain.fqdn
  }
  output "node1_zmachine1_ip" {
      value = grid_deployment.d1.vms[0].ip
  }

  output "ygg_ip" {
      value = grid_deployment.d1.vms[0].ygg_ip
  }

