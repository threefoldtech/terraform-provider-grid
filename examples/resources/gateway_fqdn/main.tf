terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "grid_fqdn_proxy" "p1" {
  node     = 11
  name     = "workloadname"
  fqdn     = "hamada1.3x0.me"
  backends = [format("http://137.184.106.152:443")]
}

output "fqdn" {
  value = grid_fqdn_proxy.p1.fqdn
}
