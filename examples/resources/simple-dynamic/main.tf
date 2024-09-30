terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "random_bytes" "server1_mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "server1_mycelium_key" {
  length = 32
}

resource "grid_scheduler" "sched" {
  # a machine for the first server instance
  requests {
    name = "server1"
    cru  = 1
    sru  = 256
    mru  = 256
  }
  # # a machine for the second server instance
  # requests {
  #   name = "server2"
  #   cru  = 1
  #   sru  = 256
  #   mru  = 256
  # }
  requests {
    name          = "balancer"
    cru           = 1
    sru           = 256
    mru           = 256
    public_config = true
  }
}

resource "grid_network" "net1" {
  nodes = distinct([
    grid_scheduler.sched.nodes["server1"],
  ])
  ip_range    = "10.1.0.0/16"
  name        = "network2"
  description = "newer network"
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["server1"]) = random_bytes.server1_mycelium_key.hex
  }
}

resource "grid_deployment" "server1" {
  node         = grid_scheduler.sched.nodes["server1"]
  network_name = grid_network.net1.name
  vms {
    name             = "firstserver"
    flist            = "https://hub.grid.tf/tf-official-apps/simple-http-server-latest.flist"
    cpu              = 1
    memory           = 256
    rootfs_size      = 51200
    entrypoint       = "/main.sh"
    mycelium_ip_seed = random_bytes.server1_mycelium_ip_seed.hex
    env_vars = {
      key   = "SSH_KEY"
      value = file("~/.ssh/id_rsa.pub")
    }
  }
}

# resource "grid_deployment" "server2" {
#   node         = grid_scheduler.sched.nodes["server2"]
#   network_name = grid_network.net1.name
#   vms {
#     name             = "secondserver"
#     flist            = "https://hub.grid.tf/tf-official-apps/simple-http-server-latest.flist"
#     cpu              = 1
#     memory           = 256
#     rootfs_size      = 51200
#     entrypoint       = "/main.sh"
#     mycelium_ip_seed = random_bytes.server2_mycelium_ip_seed.hex
#     env_vars = {
#       key   = "SSH_KEY"
#       value = file("~/.ssh/id_rsa.pub")
#     }
#   }
# }

resource "grid_fqdn_proxy" "balancer" {
  node            = grid_scheduler.sched.nodes["balancer"]
  name            = "balancer"
  fqdn            = "hamada1.3x0.me"
  backends        = [format("http://[%s]", grid_deployment.server1.vms[0].mycelium_ip)]
  tls_passthrough = false
}

output "load_balancer_domain" {
  value = grid_fqdn_proxy.balancer.fqdn
}
