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
    nodes = [1]
    ip_range = "10.1.0.0/24"
    name = local.name
    description = "newer network"
    add_wg_access = true
}
 
resource "grid_deployment" "d1" {
  name = local.name
  node= 1
  network_name = grid_network.net1.name
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/grid3_ubuntu20.04-latest.flist"
    entrypoint = "/init.sh"
    cpu = 2 
    memory = 1024
    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP ashraf@thinkpad"
    }
    planetary = true
  }
  connection {
    type     = "ssh"
    user     = "root"
    agent    = true
    host     = grid_deployment.d1.vms[0].ygg_ip
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
    value = grid_deployment.d1.vms[0].ygg_ip
}

