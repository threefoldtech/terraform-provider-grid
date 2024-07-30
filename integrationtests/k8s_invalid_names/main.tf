variable "public_key" {
  type = string
}

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

resource "random_string" "name" {
  length  = 8
  special = false
}

resource "grid_scheduler" "sched" {
  requests {
    name    = "node"
    cru     = 2
    sru     = 3 * 1024
    mru     = 6 * 1024
    farm_id = 1
  }
}


resource "grid_network" "net1" {
  nodes       = [grid_scheduler.sched.nodes["node"]]
  ip_range    = "10.1.0.0/16"
  name        = random_string.name.result
  description = "kubernetes network"
}

resource "grid_kubernetes" "k8s1" {
  network_name = grid_network.net1.name
  token        = "12345678910122"
  ssh_key      = var.public_key

  master {
    disk_size = 1
    node      = grid_scheduler.sched.nodes["node"]
    name      = "mr"
    cpu       = 2
    memory    = 2048
  }
  workers {
    disk_size = 1
    node      = grid_scheduler.sched.nodes["node"]
    name      = "w0"
    cpu       = 2
    memory    = 2048
  }
  workers {
    disk_size = 1
    node      = grid_scheduler.sched.nodes["node"]
    name      = "w0"
    cpu       = 2
    memory    = 2048
  }
}
