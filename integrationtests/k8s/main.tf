
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

locals {
  master_disk_size = 2
  master_memory    = 2048
  worker_disk_size = 2
  worker_memory    = 2048
}

resource "random_string" "name" {
  length  = 8
  special = false
}

resource "grid_scheduler" "sched" {
  requests {
    name     = "node1"
    cru      = 2
    sru      = local.master_disk_size * 1024
    mru      = local.master_memory
    distinct = true
  }

  requests {
    name     = "node2"
    cru      = 2
    sru      = local.worker_disk_size * 1024
    mru      = local.worker_memory
    distinct = true
  }
}

resource "grid_network" "net1" {
  nodes         = distinct([grid_scheduler.sched.nodes["node1"], grid_scheduler.sched.nodes["node2"]])
  ip_range      = "10.1.0.0/16"
  name          = random_string.name.result
  description   = "kubernetes network"
  add_wg_access = true
}

resource "grid_kubernetes" "k8s1" {
  network_name = grid_network.net1.name
  token        = "12345678910122"
  ssh_key      = var.public_key

  master {
    disk_size = local.master_disk_size
    node      = grid_scheduler.sched.nodes["node1"]
    name      = "mr"
    cpu       = 2
    memory    = local.master_memory
    planetary = true
  }
  workers {
    disk_size = local.worker_disk_size
    node      = grid_scheduler.sched.nodes["node2"]
    name      = "w0"
    cpu       = 2
    memory    = local.worker_memory
    planetary = true
  }
}

output "mr_ygg_ip" {
  value = grid_kubernetes.k8s1.master[0].planetary_ip
}

output "wg_config" {
  value = grid_network.net1.access_wg_config
}

output "worker_ygg_ip" {
  value = grid_kubernetes.k8s1.workers[0].planetary_ip
}
