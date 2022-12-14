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
  name = "mydisk"
}

resource "grid_capacity_reserver" "cap1" {
  farm = 1
  hdd  = 12

}

resource "grid_deployment" "d1" {
  name        = local.name
  capacity_id = grid_capacity_reserver.cap1.capacity_id

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
