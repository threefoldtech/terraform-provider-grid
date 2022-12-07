terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}


resource "grid_capacity_reserver" "reserver" {
  farm   = 1
  public = true
}
resource "grid_fqdn_proxy" "p1" {
  capacity_reservation_contract_id = grid_capacity_reserver.reserver.capacity_contract_id
  name = "workloadname"
  fqdn = "remote.omar.grid.tf"
  backends = [format("https://137.184.106.152:443")]
  tls_passthrough = true
}

output "fqdn" {
    value = grid_fqdn_proxy.p1.fqdn
}
