
variable "public_key" {
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
    name             = "node1"
    cru              = 2
    sru              = 512
    mru              = 1024
    public_config    = true
    farm_id          = 1
    public_ips_count = 1
  }
}

resource "grid_network" "net1" {
  nodes       = [grid_scheduler.scheduler.nodes["node1"]]
  ip_range    = "10.1.0.0/16"
  name        = "network"
  description = "newer network"
}

resource "grid_deployment" "d1" {
  node         = grid_scheduler.scheduler.nodes["node1"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-20.04.flist"
    cpu        = 2
    publicip   = true
    memory     = 1024
    entrypoint = "/init.sh"
    env_vars = {
      SSH_KEY  = "${var.public_key}"
      TEST_VAR = "this value for test"
    }
  }
}

output "vm_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "vm_public_ip" {
  value = grid_deployment.d1.vms[0].computedip
}
