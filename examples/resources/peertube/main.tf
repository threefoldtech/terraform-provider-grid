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
    node = 7 
    name = "ashrafpeertube"
  }
  resource "grid_network" "net1" {
      nodes = [7]
      ip_range = "10.1.0.0/24"
      name = "network"
      description = "newer network"
      add_wg_access = true
  }
  resource "grid_deployment" "d1" {
    node = 7
    network_name = grid_network.net1.name
    ip_range = lookup(grid_network.net1.nodes_ip_range, 7, "")
    vms {
      name = "vm1"
      flist = "https://hub.grid.tf/omarabdul3ziz.3bot/threefoldtech-peertube-v3.0.flist"
      cpu = 2 
      publicip = true
      entrypoint = "/usr/local/bin/entrypoint.sh"
      memory = 4096
      env_vars = {
        SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP ashraf@thinkpad"
        PEERTUBE_DB_SUFFIX="_prod"
        PEERTUBE_DB_USERNAME="peertube"
        PEERTUBE_DB_PASSWORD="peertube"
        PEERTUBE_ADMIN_EMAIL="support@threefold.com"
        PEERTUBE_WEBSERVER_HOSTNAME = data.grid_gateway_domain.domain.fqdn
        PEERTUBE_WEBSERVER_PORT=443
        PEERTUBE_SMTP_HOSTNAME="https://app.sendgrid.com"
        PEERTUBE_SMTP_USERNAME="sendgridusername"
        PEERTUBE_SMTP_PASSWORD="sendgridpassword"
      }
      planetary = true
    }
  }
  resource "grid_name_proxy" "p1" {
    node = 7
    name = "ashrafpeertube"
    backends = [format("http://%s:9000", split("/", grid_deployment.d1.vms[0].computedip)[0])]
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

