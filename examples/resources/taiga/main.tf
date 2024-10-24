terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}
locals {
  solution_type = "Taiga"
  name          = "grid3taiga"
}

provider "grid" {
}

resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}

resource "grid_scheduler" "sched" {
  requests {
    name      = "node"
    cru       = 4
    sru       = 100 * 1024
    mru       = 8096
    yggdrasil = false
    wireguard = true
  }

  requests {
    name             = "gateway"
    public_config    = true
    public_ips_count = 1
    yggdrasil        = false
    wireguard        = false
  }
}

resource "grid_network" "net2" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = [grid_scheduler.sched.nodes["node"]]
  ip_range      = "10.1.0.0/16"
  description   = "taiga network"
  add_wg_access = true
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["node"]) = random_bytes.mycelium_key.hex
  }
}

resource "grid_deployment" "node1" {
  solution_type = local.solution_type
  name          = local.name
  node          = grid_scheduler.sched.nodes["node"]
  network_name  = grid_network.net2.name
  disks {
    name = "data0"
    # will hold images, volumes etc. modify the size according to your needs
    size        = 100
    description = "volume holding docker data"
  }
  vms {
    name        = "taiga"
    flist       = "https://hub.grid.tf/tf-official-apps/grid3_taiga_docker-latest.flist"
    entrypoint  = "/sbin/zinit init"
    cpu         = 4
    memory      = 8096
    rootfs_size = 51200
    mounts {
      name        = "data0"
      mount_point = "/var/lib/docker"
    }
    env_vars = {
      SSH_KEY        = file("~/.ssh/id_rsa.pub"),
      DOMAIN_NAME    = data.grid_gateway_domain.domain.fqdn,
      ADMIN_USERNAME = "sameh",
      ADMIN_PASSWORD = "password",
      ADMIN_EMAIL    = "samehabouelsaad@gmail.com",
      # configure smtp settings bellow only If you have an working smtp service and you know what you’re doing.
      # otherwise leave these settings empty. gives wrong smtp settings will cause issues/server errors in taiga.
      DEFAULT_FROM_EMAIL  = "",
      EMAIL_USE_TLS       = "", # either "True" or "False"
      EMAIL_USE_SSL       = "", # either "True" or "False"
      EMAIL_HOST          = "",
      EMAIL_PORT          = "",
      EMAIL_HOST_USER     = "",
      EMAIL_HOST_PASSWORD = "",
    }
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
  }
}

data "grid_gateway_domain" "domain" {
  node = grid_scheduler.sched.nodes["gateway"]
  name = "grid3taiga"
}
resource "grid_name_proxy" "p1" {
  solution_type   = local.solution_type
  name            = local.name
  node            = grid_scheduler.sched.nodes["gateway"]
  backends        = [format("http://[%s]:9000", grid_deployment.node1.vms[0].mycelium_ip)]
  tls_passthrough = false
}

output "node1_zmachine1_ip" {
  value = grid_deployment.node1.vms[0].ip
}

output "node1_zmachine1_mycelium_ip" {
  value = grid_deployment.node1.vms[0].mycelium_ip
}

output "fqdn" {
  value = data.grid_gateway_domain.domain.fqdn
}
