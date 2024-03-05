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
  requests {
    name             = "master_node"
    cru              = 2
    sru              = 512
    mru              = 2048
    distinct         = true
    public_ips_count = 1
    public_config    = true
  }
  requests {
    name     = "worker1_node"
    cru      = 2
    sru      = 512
    mru      = 2048
    distinct = true
  }
  requests {
    name     = "worker2_node"
    cru      = 2
    sru      = 512
    mru      = 2048
    distinct = true
  }
  requests {
    name     = "worker3_node"
    cru      = 2
    sru      = 512
    mru      = 2048
    distinct = true
  }
}

locals {
  solution_type = "Kubernetes"
  name          = "myk8s"
}
resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = distinct(values(grid_scheduler.sched.nodes))
  ip_range      = "10.1.0.0/16"
  description   = "newer network"
  add_wg_access = true
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["master_node"])  = "9751c596c7c951aedad1a5f78f18b59515064adf660e0d55abead65e6fbbd627"
    format("%s", grid_scheduler.sched.nodes["worker1_node"]) = "d88204d7c80f98bf6ddd62cdef5e6572e5f67a1d5b8db404880d6a063797956d"
    format("%s", grid_scheduler.sched.nodes["worker2_node"]) = "01e92113d4d9fc12bd7980548b62e2bb548cebfb00529f122b76fc0768d4f65c"
    format("%s", grid_scheduler.sched.nodes["worker3_node"]) = "247933aec8bfdc658c96ce0aa7987a76681b4b9d5759437253381ed65f46a4ed"
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
    mycelium_ip_seed = "b60f2b7ec39c"
  }
  workers {
    disk_size        = 2
    node             = grid_scheduler.sched.nodes["worker1_node"]
    name             = "w0"
    cpu              = 2
    memory           = 2048
    mycelium_ip_seed = "9f50592d6b55"
  }
  workers {
    disk_size        = 2
    node             = grid_scheduler.sched.nodes["worker2_node"]
    name             = "w2"
    cpu              = 2
    memory           = 2048
    mycelium_ip_seed = "d04c42aa2a1a"
  }
  workers {
    disk_size        = 2
    node             = grid_scheduler.sched.nodes["worker3_node"]
    name             = "w3"
    cpu              = 2
    memory           = 2048
    mycelium_ip_seed = "60a9601d738d"
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

