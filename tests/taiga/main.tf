
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

resource "grid_network" "net2" {
  nodes         = [2]
  ip_range      = "10.1.0.0/16"
  name          = "network1"
  description   = "newer network"
  add_wg_access = true
}

resource "grid_deployment" "node1" {
  node         = 2
  network_name = grid_network.net2.name
  disks {
    name        = "data0"
    # will hold images, volumes etc. modify the size according to your needs
    size        = 20
    description = "volume holding docker data"
  }
  vms {
    name        = "taiga"
    flist       = "https://hub.grid.tf/tf-official-apps/grid3_taiga_docker-latest.flist"
    entrypoint  = "/sbin/zinit init"
    cpu         = 2
    memory      = 8096
    rootfs_size = 51200
    mounts {
      disk_name   = "data0"
      mount_point = "/var/lib/docker"
    }
    env_vars = {
      TEST_VAR = "this value for test"
      SSH_KEY = "${var.public_key}",
      DOMAIN_NAME = data.grid_gateway_domain.domain.fqdn,
      ADMIN_USERNAME = "khaled",
      ADMIN_PASSWORD = "password",
      ADMIN_EMAIL = "khaledgx96@gmail.com",
      # configure smtp settings bellow only If you have an working smtp service and you know what you’re doing.
      # otherwise leave these settings empty. gives wrong smtp settings will cause issues/server errors in taiga.
      DEFAULT_FROM_EMAIL = "",
      EMAIL_USE_TLS = "", # either "True" or "False"
      EMAIL_USE_SSL = "", # either "True" or "False"
      EMAIL_HOST = "",
      EMAIL_PORT = "",
      EMAIL_HOST_USER = "",
      EMAIL_HOST_PASSWORD = "",
    }
    planetary = true
    publicip = true
  }
}

data "grid_gateway_domain" "domain" {
  node = 2
  name = "grid3taiga"
}
resource "grid_name_proxy" "p1" {
  node            = 2
  name            = "grid3taiga"
  backends        = [format("http://%s:9000", grid_deployment.node1.vms[0].ygg_ip)]
  tls_passthrough = false
}

output "node1_zmachine1_ip" {
  value = grid_deployment.node1.vms[0].ip
}


output "node1_zmachine1_ygg_ip" {
  value = grid_deployment.node1.vms[0].ygg_ip
}

output "fqdn" {
  value = data.grid_gateway_domain.domain.fqdn
}
