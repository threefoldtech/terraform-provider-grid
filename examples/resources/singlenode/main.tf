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

resource "grid_network" "net1" {
    nodes = [2]
    ip_range = "172.20.0.0/16"
    name = "net1"                       
    description = "new network"
}


resource "grid_deployment" "d1" {
  node = 2
  network_name = grid_network.net1.name
  ip_range = grid_network.net1.nodes_ip_range[2]

  vms {
    name = "pt"
    flist = "https://hub.grid.tf/omar0.3bot/omarelawady-peertube-latest.flist"
    cpu = 1
    publicip = true
    memory = 1024
    entrypoint = "/usr/local/bin/startup.sh example.gent01.devnet.grid.tf"
    planetary = true
  }
}


resource "grid_proxy" "p1" {
  node = 2
  name = "example"
  backends = [format("http://%s", trimsuffix(grid_deployment.d1.vms[0].computedip, "/24"))]
  
}
output "fqdn" {
    value = grid_proxy.p1.fqdn
}

output "ygg_ip" {
    value = grid_deployment.d1.vms[0].ygg_ip
}

output "ygg_ip" {
    value = grid_deployment.d1.vms[1].ygg_ip
}
