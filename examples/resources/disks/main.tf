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
    name = "node1"
    sru  = 1024 * 12
  }
}

locals {
  name  = "mydisk"
  node1 = grid_scheduler.sched.nodes["node1"]
}

resource "grid_deployment" "d1" {
  name = local.name
  node = local.node1

  zdbs {
    name        = "zdb1"
    size        = 10
    description = "zdb1 description"
    password    = "zdbpasswd1"
    mode        = "user"
  }
  zdbs {
    name        = "zdb2"
    size        = 2
    description = "zdb2 description"
    password    = "zdbpasswd2"
    mode        = "seq"
  }
}

output "deployment_id" {
  value = grid_deployment.d1.id
}

output "zdb1_endpoint" {
  value = format("[%s]:%d", grid_deployment.d1.zdbs[0].ips[0], grid_deployment.d1.zdbs[0].port)
}

output "zdb1_namespace" {
  value = grid_deployment.d1.zdbs[0].namespace
}

output "zdb1_mycelium_ip" {
  value = grid_deployment.d1.zdbs[0].ips[2]
}
