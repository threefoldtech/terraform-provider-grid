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
  # a machine for the first server instance
  requests {
    name = "server1"
    cru  = 1
    sru  = 256
    mru  = 256
  }
  # a machine for the second server instance
  requests {
    name = "server2"
    cru  = 1
    sru  = 256
    mru  = 256
  }

}

resource "grid_network" "net1" {
  nodes = distinct([
    grid_scheduler.sched.nodes["server1"],
    grid_scheduler.sched.nodes["server2"]
  ])
  ip_range    = "10.1.0.0/16"
  name        = "network2"
  description = "newer network"
}

resource "grid_deployment" "server1" {
  node         = grid_scheduler.sched.nodes["server1"]
  network_name = grid_network.net1.name
  vms {
    name        = "firstserver"
    flist       = "https://hub.grid.tf/tf-official-apps/simple-http-server-latest.flist"
    cpu         = 1
    memory      = 256
    rootfs_size = 51200
    entrypoint  = "/main.sh"
    env_vars = {
      key   = "SSH_KEY"
      value = file("~/.ssh/id_rsa.pub")
    }

    planetary = true
  }
}

resource "grid_deployment" "server2" {
  node         = grid_scheduler.sched.nodes["server2"]
  network_name = grid_network.net1.name
  vms {
    name        = "secondserver"
    flist       = "https://hub.grid.tf/tf-official-apps/simple-http-server-latest.flist"
    cpu         = 1
    memory      = 256
    rootfs_size = 51200
    entrypoint  = "/main.sh"
    env_vars = {
      key   = "SSH_KEY"
      value = file("~/.ssh/id_rsa.pub")
    }

    planetary = true
  }
}

resource "grid_fqdn_proxy" "balancer" {
  node            = 15
  name            = "balancer"
  fqdn            = "hamada1.3x0.me"
  backends        = [format("http://[%s]", grid_deployment.server1.vms[0].planetary_ip), format("http://[%s]", grid_deployment.server2.vms[0].planetary_ip)]
  tls_passthrough = false
}


output "load_balancer_domain" {
  value = grid_fqdn_proxy.balancer.fqdn
}
