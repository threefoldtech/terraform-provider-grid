variable "fqdn" {
  type = string
}

variable "backend" {
  type = string
}

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
  node = 1
  name = "testname"
  fqdn = "${var.fqdn}"
  backends = [format("${var.backend}")]
  tls_passthrough = false
}

output "fqdn" {
    value = grid_fqdn_proxy.p1.fqdn
}
