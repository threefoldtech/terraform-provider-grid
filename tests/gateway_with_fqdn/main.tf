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
      version = "0.1.2"
    }
  }
}

provider "grid" {
}

resource "grid_fqdn_proxy" "p1" {
  node = 5
  name = "testname"
  fqdn = "${var.fqdn}"
  backends = [format("${var.fqdn}")]
  tls_passthrough = true
}

output "fqdn" {
    value = grid_fqdn_proxy.p1.fqdn
}
