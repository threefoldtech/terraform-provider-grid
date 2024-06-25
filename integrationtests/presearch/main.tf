
variable "public_key" {
  type = string
}

variable "presearch_registration_code" {
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

locals {
  solution_type = "Presearch"
  name          = "presearch"
}


resource "grid_scheduler" "sched" {
  requests {
    name = "presearch"
    cru  = 1
    sru  = 5 * 1024
    mru  = 1024
  }
}

resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = [grid_scheduler.sched.nodes["presearch"]]
  ip_range      = "10.1.0.0/16"
  description   = "presearch network"
}

# Deployment specs
resource "grid_deployment" "d1" {
  solution_type = local.solution_type
  name          = local.name
  node          = grid_scheduler.sched.nodes["presearch"]
  network_name  = grid_network.net1.name

  disks {
    name        = "data"
    size        = 5
    description = "volume holding docker data"
  }

  vms {
    name       = local.name
    flist      = "https://hub.grid.tf/tf-official-apps/presearch-v2.2.flist"
    entrypoint = "/sbin/zinit init"
    planetary  = true
    cpu        = 1
    memory     = 1024

    mounts {
      disk_name   = "data"
      mount_point = "/var/lib/docker"
    }

    env_vars = {
      SSH_KEY                     = "${var.public_key}",
      PRESEARCH_REGISTRATION_CODE = "e5083a8d0a6362c6cf7a3078bfac81e3",

    }
  }
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].planetary_ip
}
