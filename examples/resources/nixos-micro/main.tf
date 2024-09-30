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
  name    = "myvm"
  node_id = 11
}

resource "grid_network" "net1" {
  name        = local.name
  nodes       = [local.node_id]
  ip_range    = "10.1.0.0/16"
  description = "newer network"
  # add_wg_access = true
  mycelium_keys = {
    format("%s", local.node_id) = random_bytes.mycelium_key.hex
  }
}
resource "grid_deployment" "d1" {
  node         = local.node_id
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
      name        = "store"
      mount_point = "/nix"
    }
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex

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

output "mycelium_ip" {
  value = grid_deployment.d1.vms[0].mycelium_ip
}
