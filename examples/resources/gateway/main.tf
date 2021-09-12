terraform {
  required_providers {
    grid = {
      source = "threefoldtechdev.com/providers/grid"
      version = "0.2"
    }
  }
}

provider "grid" {
}


resource "grid_proxy" "p1" {
  node = 2
  name = "example"
  backends = [format("http://69.164.223.208")]
  
}
output "fqdn" {
    value = grid_proxy.p1.fqdn
}
