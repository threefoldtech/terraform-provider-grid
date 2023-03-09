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
    name     = "master_node"
    cru      = 2
    sru      = 512
    mru      = 2048
    distinct = true
    public_ips_count = 1
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
}

resource "grid_kubernetes" "k8s1" {
  solution_type = local.solution_type
  name          = local.name
  network_name  = grid_network.net1.name
  token         = "12345678910122"
  ssh_key       = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP ashraf@thinkpad"

  master {
    disk_size = 2
    node      = grid_scheduler.sched.nodes["master_node"]
    name      = "mr"
    cpu       = 2
    publicip  = true
    memory    = 2048
  }
  workers {
    disk_size = 2
    node      = grid_scheduler.sched.nodes["worker1_node"]
    name      = "w0"
    cpu       = 2
    memory    = 2048
  }
  workers {
    disk_size = 2
    node      = grid_scheduler.sched.nodes["worker2_node"]
    name      = "w2"
    cpu       = 2
    memory    = 2048
  }
  workers {
    disk_size = 2
    node      = grid_scheduler.sched.nodes["worker3_node"]
    name      = "w3"
    cpu       = 2
    memory    = 2048
  }
}

output "computed_master_public_ip" {
  value = grid_kubernetes.k8s1.master[0].computedip
}

output "wg_config" {
  value = grid_network.net1.access_wg_config
}
