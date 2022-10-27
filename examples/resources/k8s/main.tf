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
  solution_type = "Kubernetes"
  name          = "myk8s"
}

resource "grid_capacity_reserver" "deployer1" {
  network = "net1"
  nodes   = [34, 33, 49]
  deployments {
    farm   = "1"
    node   = 34
    cpu    = 2
    memory = 1024

  }
  deployments {
    farm   = "1"
    node   = 33
    cpu    = 4
    memory = 1024
  }
  deployments {
    farm   = "1"
    node   = 49
    cpu    = 4
    memory = 1024
  }

}
resource "grid_network" "net1" {
  solution_type = local.solution_type
  nodes         = grid_capacity_reserver.deployer1.nodes
  ip_range      = "10.1.0.0/16"
  name          = local.name
  description   = "newer network"
  add_wg_access = true
}

resource "grid_kubernetes" "k8s1" {
  solution_type  = local.solution_type
  name           = local.name
  network_name   = grid_network.net1.name
  nodes_ip_range = grid_network.net1.nodes_ip_range
  token          = "12345678910122"
  ssh_key        = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP ashraf@thinkpad"

  master {
    disk_size = 23
    node      = grid_capacity_reserver.deployer1.deployments[0].node
    name      = "mr"
    cpu       = 2
    planetary = true
    memory    = 2048
  }
  workers {
    disk_size = 15
    node      = grid_capacity_reserver.deployer1.deployments[1].node
    name      = "w0"
    cpu       = 2
    memory    = 2048
  }
  workers {
    disk_size = 14
    node      = grid_capacity_reserver.deployer1.deployments[2].node
    name      = "w2"
    cpu       = 1
    memory    = 2048
  }

}


output "master_public_ip" {
  value = grid_kubernetes.k8s1.master[0].computedip
}

output "wg_config" {
  value = grid_network.net1.access_wg_config
}
