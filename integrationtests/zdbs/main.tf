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
    name = "node1"
    hru  = 10240
  }
}

resource "grid_deployment" "d1" {
  node = grid_scheduler.scheduler.nodes["node1"]

  zdbs {
    name        = "zdb123"
    size        = 10
    description = "zdb1 description"
    password    = var.password
    mode        = "user"
  }
}


output "deployment_id" {
  value = grid_deployment.d1.id
}

output "zdb1_endpoint" {
  value = format("[%s]:%d", grid_deployment.d1.zdbs[0].ips[1], grid_deployment.d1.zdbs[0].port)
}

output "zdb1_namespace" {
  value = grid_deployment.d1.zdbs[0].namespace
}
