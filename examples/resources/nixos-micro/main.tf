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
  name        = local.name
  nodes       = [34]
  ip_range    = "10.1.0.0/16"
  description = "newer network"
  # add_wg_access = true
}
resource "grid_deployment" "d1" {
  node         = 34
  network_name = grid_network.net1.name
  disks {
    name        = "store"
    size        = 50
    description = "volume holding store data"
  }

  vms {
    name  = "vm1"
    flist = "https://hub.grid.tf/tf-official-vms/nixos-micro-latest.flist"
    cpu   = 2
    # publicip   = true
    memory     = 2048
    entrypoint = "/entrypoint.sh"
    mounts {
      disk_name   = "store"
      mount_point = "/nix"
    }

    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")

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

output "wg_config" {
  value = grid_network.net1.access_wg_config
}
output "node1_zmachine1_ip" {
  value = grid_deployment.d1.vms[0].ip
}
output "computed_public_ip" {
  value = split("/", grid_deployment.d1.vms[0].computedip)[0]
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].planetary_ip
}
