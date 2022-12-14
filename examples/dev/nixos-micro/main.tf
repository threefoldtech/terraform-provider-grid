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
resource "grid_capacity_reserver" "cap1" {
  farm   = 1
  cpu    = 2
  memory = 2048
  ssd    = 50

}

resource "grid_network" "net1" {
  name         = local.name
  capacity_ids = [grid_capacity_reserver.cap1.capacity_id]
  ip_range     = "10.1.0.0/16"
  description  = "newer network"
  # add_wg_access = true
}
resource "grid_deployment" "d1" {
  capacity_id  = grid_capacity_reserver.cap1.capacity_id
  network_name = grid_network.net1.name
  disks {
    name        = "store"
    size        = 50
    description = "volume holding store data"
  }

  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-vms/nixos-micro-latest.flist"
    cpu        = 2
    publicip   = true
    memory     = 2048
    entrypoint = "/entrypoint.sh"
    mounts {
      disk_name   = "store"
      mount_point = "/nix"
    }

    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP ashraf@thinkpad"

      NIX_CONFIG = <<EOT
{ pkgs ? import <nixpkgs> { }, pythonPackages ? pkgs.python3Packages }:

pkgs.mkShell {
  buildInputs = [
     pythonPackages.numpy
     pythonPackages.scipy
     pythonPackages.jupyterlab
  ];

}
EOT
    }

    planetary = true
  }
}

output "node1_zmachine1_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "public_ip" {
  value = grid_deployment.d1.vms[0].computedip
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].ygg_ip
}
