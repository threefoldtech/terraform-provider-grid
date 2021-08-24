terraform {
  required_providers {
    grid = {
      version = "0.1"
      source  = "threefoldtech.com/providers/grid"
    }
  }
}

provider "grid" {    
}

resource "grid_network" "net1" {
    nodes = [2, 3]
    ip_range = "10.1.0.0/16"
    name = "network"
    description = "very newer network"
}


output "wg_config" {
    value = grid_network.net1.access_wg_config
}
