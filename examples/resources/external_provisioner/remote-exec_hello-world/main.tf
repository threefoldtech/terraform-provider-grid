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
  name = "myvm"
}

resource "grid_network" "net1" {
  nodes         = [1]
  ip_range      = "10.1.0.0/24"
  name          = local.name
  description   = "newer network"
  add_wg_access = true
}

resource "grid_deployment" "d1" {
  name         = local.name
  node         = 1
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
    planetary = true
  }
  connection {
    type  = "ssh"
    user  = "root"
    agent = true
    host  = grid_deployment.d1.vms[0].planetary_ip
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

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].planetary_ip
}

