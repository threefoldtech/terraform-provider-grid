terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}
locals {
  solution_type = "Kubernetes"
  name          = "mykubernetes"
}
resource "group" "g1" {

}
resource "grid_capacity_reserver" "master" {
  farm     = 1
  cpu      = 2
  memory   = 2048
  ssd      = 25
  group_id = group.g1.group_id
  public   = true

}
resource "grid_capacity_reserver" "worker1" {
  farm     = 1
  cpu      = 2
  memory   = 2048
  ssd      = 15
  group_id = group.g1.group_id

}
resource "grid_capacity_reserver" "worker2" {
  farm     = 1
  cpu      = 2
  memory   = 2048
  ssd      = 15
  group_id = group.g1.group_id

}


resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = [grid_capacity_reserver.master.capacity_id, grid_capacity_reserver.worker1.capacity_id, grid_capacity_reserver.worker2.capacity_id]
  ip_range      = "10.1.0.0/16"
  description   = "newer network"
  add_wg_access = true
}

resource "grid_kubernetes" "k8s1" {
  solution_type  = local.solution_type
  name           = local.name
  network_name   = grid_network.net1.name
  nodes_ip_range = grid_network.net1.nodes_ip_range
  token          = "12345678910122"
  ssh_key        = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP"

  master {
    disk_size   = 25
    capacity_id = grid_capacity_reserver.master.capacity_id
    name        = "mr"
    cpu         = 2
    publicip    = true
    memory      = 2048
  }
  workers {
    disk_size   = 15
    capacity_id = grid_capacity_reserver.worker1.capacity_id
    name        = "w0"
    cpu         = 2
    memory      = 2048
  }
  workers {
    disk_size   = 15
    capacity_id = grid_capacity_reserver.worker2.capacity_id
    name        = "w2"
    cpu         = 2
    memory      = 2048
  }

}

data "grid_gateway_domain" "domain" {
  node = grid_capacity_reserver.master.capacity_id
  name = "ashraf"
}
resource "grid_name_proxy" "p1" {
  capacity_id     = grid_capacity_reserver.master.capacity_id
  name            = "ashraf"
  backends        = [format("https://%s:443", split("/", grid_kubernetes.k8s1.master[0].computedip)[0])]
  tls_passthrough = true
}
output "master_public_ip" {
  value = grid_kubernetes.k8s1.master[0].computedip
}
output "fqdn" {
  value = data.grid_gateway_domain.domain.fqdn
}
output "wg_config" {
  value = grid_network.net1.access_wg_config
}
