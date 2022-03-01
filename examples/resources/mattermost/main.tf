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
    node = 8
    name = "ashrafmattermost"
  }
  resource "grid_network" "net1" {
      nodes = [8]
      ip_range = "10.1.0.0/16"
      name = "networkk"
      description = "newer network"
      add_wg_access = true
  }
  resource "grid_deployment" "d1" {
    node = 8
    network_name = grid_network.net1.name
    ip_range = lookup(grid_network.net1.nodes_ip_range, 8, "")
    vms {
      name = "vm1"
      flist = "https://hub.grid.tf/ashraf.3bot/ashraffouda-mattermost-latest.flist"
      cpu = 2 
      entrypoint = "/sbin/zinit init"
      memory = 4096
      env_vars = {
        SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP ashraf@thinkpad"
        DB_PASSWORD = "ashroof"
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
    node = 8
    name = "ashrafmattermost"
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
