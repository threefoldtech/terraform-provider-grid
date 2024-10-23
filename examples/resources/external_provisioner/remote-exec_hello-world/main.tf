terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}


locals {
  name = "myvm"
}

resource "grid_scheduler" "sched" {
  requests {
    name         = "node1"
    cru          = 2
    sru          = 1024
    mru          = 1024
    node_exclude = [33] # exlude node 33 from your search
    yggdrasil    = false
    wireguard    = true
  }
}

resource "grid_network" "net1" {
  nodes         = [grid_scheduler.sched.nodes["node1"]]
  ip_range      = "10.1.0.0/16"
  name          = local.name
  description   = "newer network"
  add_wg_access = true
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["node1"]) = random_bytes.mycelium_key.hex
  }
}

resource "grid_deployment" "d1" {
  name         = local.name
  node         = grid_scheduler.sched.nodes["node1"]
  network_name = grid_network.net1.name
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/grid3_ubuntu20.04-latest.flist"
    entrypoint = "/init.sh"
    cpu        = 2
    memory     = 1024
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
  }
  connection {
    type  = "ssh"
    user  = "root"
    agent = true
    host  = grid_deployment.d1.vms[0].mycelium_ip
  }

  provisioner "remote-exec" {
    inline = [
      "echo 'Hello world!' > /root/readme.txt"
    ]
  }
}

output "node1_zmachine1_ip" {
  value = grid_deployment.d1.vms[0].ip
}

output "mycelium_ip" {
  value = grid_deployment.d1.vms[0].mycelium_ip
}
