 terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "grid_network" "net1" {
    nodes = [7]
    ip_range = "10.1.0.0/24"
    name = "network"
    description = "newer network"
    add_wg_access = true
}
 
resource "grid_deployment" "d1" {
  node = 7
  network_name = grid_network.net1.name
  ip_range = lookup(grid_network.net1.nodes_ip_range, 7, "")
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/samehabouelsaad.3bot/abouelsaad-grid3_ubuntu20.04-latest.flist"
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

