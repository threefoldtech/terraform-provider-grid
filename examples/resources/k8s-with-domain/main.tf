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

resource "grid_scheduler" "sched" {
  requests {
    name             = "node1"
    cru              = 4
    mru              = 4 * 1024
    public_config    = true
    public_ips_count = 1
  }
  requests {
    name = "node2"
    cru  = 2
    mru  = 2 * 1024
  }
  requests {
    name             = "gateway"
    public_config    = true
    public_ips_count = 1
  }
}

resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = [grid_scheduler.sched.nodes["node1"], grid_scheduler.sched.nodes["node2"]]
  ip_range      = "10.1.0.0/16"
  description   = "newer network"
  add_wg_access = true
}

resource "grid_kubernetes" "k8s1" {
  solution_type = local.solution_type
  name          = local.name
  network_name  = grid_network.net1.name
  token         = "12345678910122"
  ssh_key       = file("~/.ssh/id_rsa.pub")

  master {
    disk_size = 23
    node      = grid_scheduler.sched.nodes["node1"]
    name      = "mr"
    cpu       = 2
    publicip  = true
    memory    = 2048
  }
  workers {
    disk_size = 15
    node      = grid_scheduler.sched.nodes["node1"]
    name      = "w0"
    cpu       = 2
    memory    = 2048
  }
  workers {
    disk_size = 14
    node      = grid_scheduler.sched.nodes["node2"]
    name      = "w2"
    cpu       = 1
    memory    = 2048
  }
  workers {
    disk_size = 13
    node      = grid_scheduler.sched.nodes["node2"]
    name      = "w3"
    cpu       = 1
    memory    = 2048
  }
}

data "grid_gateway_domain" "domain" {
  node = grid_scheduler.sched.nodes["gateway"]
  name = "ashraf"
}
resource "grid_name_proxy" "p1" {
  node     = grid_scheduler.sched.nodes["gateway"]
  name     = "ashraf"
  backends = [format("http://%s:443", split("/", grid_kubernetes.k8s1.master[0].computedip)[0])]
}
output "computed_master_public_ip" {
  value = grid_kubernetes.k8s1.master[0].computedip
}
output "fqdn" {
  value = data.grid_gateway_domain.domain.fqdn
}
output "wg_config" {
  value = grid_network.net1.access_wg_config
}
