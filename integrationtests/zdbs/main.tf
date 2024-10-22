variable "password" {
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

resource "grid_scheduler" "scheduler" {
  requests {
    name = "node"
    hru  = 10 * 1024
  }
}

resource "random_string" "name" {
  length  = 8
  special = false
}

resource "grid_deployment" "d1" {
  node = grid_scheduler.scheduler.nodes["node"]

  zdbs {
    name        = random_string.name.result
    size        = 10
    description = "zdb description"
    password    = var.password
    mode        = "user"
  }
}

output "deployment_id" {
  value = grid_deployment.d1.id
}

output "zdb1_endpoint" {
  value = format("[%s]:%d", grid_deployment.d1.zdbs[0].ips[2], grid_deployment.d1.zdbs[0].port)
}

output "zdb1_namespace" {
  value = grid_deployment.d1.zdbs[0].namespace
}
