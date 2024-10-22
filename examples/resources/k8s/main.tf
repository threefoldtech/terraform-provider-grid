terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "random_bytes" "master_mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "worker1_mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "worker2_mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "worker3_mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "master_mycelium_key" {
  length = 32
}

resource "random_bytes" "worker1_mycelium_key" {
  length = 32
}

resource "random_bytes" "worker2_mycelium_key" {
  length = 32
}

resource "random_bytes" "worker3_mycelium_key" {
  length = 32
}

resource "grid_scheduler" "sched" {
  requests {
    name             = "master_node"
    cru              = 2
    sru              = 512
    mru              = 2048
    distinct         = true
    public_ips_count = 1
    public_config    = true
    yggdrasil        = false
    wireguard        = true
  }
  requests {
    name      = "worker1_node"
    cru       = 2
    sru       = 512
    mru       = 2048
    distinct  = true
    yggdrasil = false
    wireguard = true
  }
  requests {
    name      = "worker2_node"
    cru       = 2
    sru       = 512
    mru       = 2048
    distinct  = true
    yggdrasil = false
    wireguard = true
  }
  requests {
    name      = "worker3_node"
    cru       = 2
    sru       = 512
    mru       = 2048
    distinct  = true
    yggdrasil = false
    wireguard = true
  }
}

locals {
  solution_type = "kubernetes/mr"
  name          = "myk8s"
}

resource "grid_network" "net1" {
  name          = local.name
  nodes         = distinct(values(grid_scheduler.sched.nodes))
  ip_range      = "10.1.0.0/16"
  description   = "kubernetes network"
  add_wg_access = true
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["master_node"])  = random_bytes.master_mycelium_key.hex
    format("%s", grid_scheduler.sched.nodes["worker1_node"]) = random_bytes.worker1_mycelium_key.hex
    format("%s", grid_scheduler.sched.nodes["worker2_node"]) = random_bytes.worker2_mycelium_key.hex
    format("%s", grid_scheduler.sched.nodes["worker3_node"]) = random_bytes.worker3_mycelium_key.hex
  }
}

resource "grid_kubernetes" "k8s1" {
  solution_type = local.solution_type
  name          = local.name
  network_name  = grid_network.net1.name
  token         = "12345678910122"
  ssh_key       = file("~/.ssh/id_rsa.pub")

  master {
    disk_size        = 2
    node             = grid_scheduler.sched.nodes["master_node"]
    name             = "mr"
    cpu              = 2
    publicip         = true
    memory           = 2048
    mycelium_ip_seed = random_bytes.master_mycelium_ip_seed.hex
  }
  workers {
    disk_size        = 2
    node             = grid_scheduler.sched.nodes["worker1_node"]
    name             = "w0"
    cpu              = 2
    memory           = 2048
    mycelium_ip_seed = random_bytes.worker1_mycelium_ip_seed.hex
  }
  workers {
    disk_size        = 2
    node             = grid_scheduler.sched.nodes["worker2_node"]
    name             = "w2"
    cpu              = 2
    memory           = 2048
    mycelium_ip_seed = random_bytes.worker2_mycelium_ip_seed.hex
  }
  workers {
    disk_size        = 2
    node             = grid_scheduler.sched.nodes["worker3_node"]
    name             = "w3"
    cpu              = 2
    memory           = 2048
    mycelium_ip_seed = random_bytes.worker3_mycelium_ip_seed.hex
  }
}

output "computed_master_public_ip" {
  value = grid_kubernetes.k8s1.master[0].computedip
}

output "computed_master_mycelium_ip" {
  value = grid_kubernetes.k8s1.master[0].mycelium_ip
}

output "computed_worker1_mycelium_ip" {
  value = grid_kubernetes.k8s1.workers[0].mycelium_ip
}

output "computed_worker2_mycelium_ip" {
  value = grid_kubernetes.k8s1.workers[1].mycelium_ip
}

output "computed_worker3_mycelium_ip" {
  value = grid_kubernetes.k8s1.workers[2].mycelium_ip
}

output "wg_config" {
  value = grid_network.net1.access_wg_config
}

output "master_console_url" {
  value = grid_kubernetes.k8s1.master[0].console_url
}

