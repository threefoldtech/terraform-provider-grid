terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "grid_scheduler" "sched" {
  requests {
    name             = "gateway"
    public_config    = true
    public_ips_count = 1
  }
}

resource "grid_fqdn_proxy" "p1" {
  node     = grid_scheduler.sched.nodes["gateway"]
  name     = "workloadname"
  fqdn     = "remote.omar.grid.tf"
  backends = [format("http://137.184.106.152:443")]
}

output "fqdn" {
  value = grid_fqdn_proxy.p1.fqdn
}
