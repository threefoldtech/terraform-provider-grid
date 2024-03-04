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
  name = "testvm"
}

resource "grid_scheduler" "sched" {
  requests {
    name             = "node1"
    cru              = 3
    sru              = 1024
    mru              = 2048
    node_exclude     = [33]  # exlude node 33 from your search
    public_ips_count = 0     # this deployment needs 0 public ips
    public_config    = false # this node does not need to have public config
  }
}

resource "grid_network" "net1" {
  name        = local.name
  nodes       = [grid_scheduler.sched.nodes["node1"]]
  ip_range    = "10.1.0.0/16"
	mycelium_keys = {
		format("%s", grid_scheduler.sched.nodes["node1"]) = "9751c596c7c951aedad1a5f78f18b59515064adf660e0d55abead65e6fbbd627"
	}
  description = "newer network"
}
resource "grid_deployment" "d1" {
  name         = local.name
  node         = grid_scheduler.sched.nodes["node1"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 2
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
    # planetary = true
		mycelium_ip_seed = "b60f2b7ec39c"
  }
}
output "vm1_ip" {
  value = grid_deployment.d1.vms[0].ip
}

output "vm1_mycelium_ip" {
  value = grid_deployment.d1.vms[0].mycelium_ip
}

output "vm1_console_url" {
  value = grid_deployment.d1.vms[0].console_url
}