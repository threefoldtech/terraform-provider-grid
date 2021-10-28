terraform {
  required_providers {
    grid = {
      source = "threefoldtechdev.com/providers/grid"
    }
  }
}

provider "grid" {
}

resource "grid_network" "net1" {
    nodes = [7]
    ip_range = "10.1.0.0/16"
    name = "network"
    description = "newer network"
    add_wg_access = false 
}
