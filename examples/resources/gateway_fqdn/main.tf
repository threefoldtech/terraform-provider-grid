terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
      version = "0.1.8"
    }
  }
}

provider "grid" {
}


resource "grid_fqdn_proxy" "p1" {
  node = 40
  name = "workloadname"
  fqdn = "remote.omar.grid.tf"
  backends = [format("https://137.184.106.152:443")]
  tls_passthrough = true
}

output "fqdn" {
    value = grid_fqdn_proxy.p1.fqdn
}
