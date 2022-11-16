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
resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = [2, 4]
  ip_range      = "10.1.0.0/16"
  description   = "newer network"
  add_wg_access = true
}

resource "grid_kubernetes" "k8s1" {
  solution_type  = local.solution_type
  name           = local.name
  network_name   = grid_network.net1.name
  token          = "12345678910122"
  ssh_key        = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP"

  master {
    disk_size = 23
    node      = 2
    name      = "mr"
    cpu       = 2
    publicip  = true
    memory    = 2048
  }
  workers {
    disk_size = 15
    node      = 2
    name      = "w0"
    cpu       = 2
    memory    = 2048
  }
  workers {
    disk_size = 14
    node      = 4
    name      = "w2"
    cpu       = 1
    memory    = 2048
  }
  workers {
    disk_size = 13
    node      = 4
    name      = "w3"
    cpu       = 1
    memory    = 2048
  }
}

data "grid_gateway_domain" "domain" {
  node = 7
  name = "ashraf"
}
resource "grid_name_proxy" "p1" {
  node            = 7
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
