
variable "public_key" {
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

resource "grid_network" "net1" {
  nodes         = [5, 7]
  ip_range      = "10.1.0.0/16"
  name          = "network12346"
  description   = "newer network"
  add_wg_access = true
}

resource "grid_kubernetes" "k8s1" {
  network_name = grid_network.net1.name
  token        = "12345678910122"
  ssh_key      = var.public_key

  master {
    disk_size = 22
    node      = 5
    name      = "mr"
    cpu       = 2
    publicip  = false
    memory    = 2048
    planetary = true
  }
  workers {
    disk_size = 15
    node      = 7
    name      = "w0"
    cpu       = 2
    memory    = 2048
  }
}


output "ygg_ip" {
  value = grid_kubernetes.k8s1.master[0].ygg_ip
}

output "wg_config" {
  value = grid_network.net1.access_wg_config
}
