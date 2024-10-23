
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
  mnemonic = "ring brand park runway cigar month yard venture else wealth surprise attract"
  network = "dev"
}

resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}

resource "random_string" "name" {
  length  = 8
  special = false
}

locals {
  solution_type = "Presearch"
}

resource "grid_scheduler" "sched" {
  requests {
    name      = "presearch"
    cru       = 1
    sru       = 5 * 1024
    mru       = 1024
    farm_id   = 1
    yggdrasil = true
    wireguard = false
  }
}

resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = random_string.name.result
  nodes         = [grid_scheduler.sched.nodes["presearch"]]
  ip_range      = "10.1.0.0/16"
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["presearch"]) = random_bytes.mycelium_key.hex
  }
  description   = "presearch network"
}

# Deployment specs
resource "grid_deployment" "d1" {
  solution_type = local.solution_type
  name          = random_string.name.result
  node          = grid_scheduler.sched.nodes["presearch"]
  network_name  = grid_network.net1.name

  disks {
    name        = "data"
    size        = 5
    description = "volume holding docker data"
  }

  vms {
    name       = random_string.name.result
    flist      = "https://hub.grid.tf/tf-official-apps/presearch-v2.2.flist"
    entrypoint = "/sbin/zinit init"
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
    cpu        = 1
    memory     = 1024

    mounts {
      name        = "data"
      mount_point = "/var/lib/docker"
    }

    env_vars = {
      SSH_KEY                     = "${var.public_key}",
      PRESEARCH_REGISTRATION_CODE = "e5083a8d0a6362c6cf7a3078bfac81e3",

    }
  }
}

output "mycelium_ip" {
  value = grid_deployment.d1.vms[0].mycelium_ip
}
