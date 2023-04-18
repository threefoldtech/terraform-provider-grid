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
resource "grid_scheduler" "sched" {
  requests {
    name = "node1"
    cru  = 2
    sru  = 512
    mru  = 2048
  }

  requests {
    name = "node2"
    cru  = 2
    sru  = 512
    mru  = 2048
  }

  requests {
    name = "node3"
    cru  = 2
    sru  = 512
    mru  = 2048
  }
}


resource "grid_network" "net1" {
  nodes = distinct([
    grid_scheduler.sched.nodes["node1"],
    grid_scheduler.sched.nodes["node2"],
    grid_scheduler.sched.nodes["node3"]
  ])
  ip_range    = "10.1.0.0/16"
  name        = "network12346"
  description = "newer network"
}
resource "grid_kubernetes" "k8s1" {
  network_name = grid_network.net1.name
  token        = "12345678910122"
  ssh_key      = var.public_key

  master {
    disk_size = 22
    node      = grid_scheduler.sched.nodes["node1"]
    name      = "mr"
    cpu       = 2
    memory    = 2048
  }
  workers {
    disk_size = 15
    node      = grid_scheduler.sched.nodes["node2"]
    name      = "w0"
    cpu       = 2
    memory    = 2048
  }
  workers {
    disk_size = 13
    node      = grid_scheduler.sched.nodes["node3"]
    name      = "w0"
    cpu       = 1
    memory    = 2048
  }
}

output "computed_master_public_ip" {
  value = grid_kubernetes.k8s1.master[0].computedip
}
