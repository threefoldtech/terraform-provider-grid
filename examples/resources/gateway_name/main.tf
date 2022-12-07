terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}


resource "grid_capacity_reserver" "deployer1" {
  farm   = 1
  public = true
}

# this data source is used to break circular dependency in cases similar to the following:
# vm: needs to know the domain in its init script
# gateway_name: needs the ip of the vm to use as backend.
# - the fqdn can be computed from grid_gateway_domain for the vm
# - the backend can reference the vm ip directly 
data "grid_gateway_domain" "domain" {
  node = grid_capacity_reserver.deployer1.node
  name = "example2"
}

resource "grid_name_proxy" "p1" {
  capacity_reservation_contract_id = grid_capacity_reserver.deployer1.capacity_contract_id
  name = "example2"
  backends = [format("http://69.164.223.208")]
  tls_passthrough = false
}
output "fqdn" {
    value = data.grid_gateway_domain.domain.fqdn
}
