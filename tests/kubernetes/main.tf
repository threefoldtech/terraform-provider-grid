
variable "public_key" {
  type = string
}

terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
      version = "0.1.1"
    }
  }
}

provider "grid" {
}

resource "grid_network" "net1" {
    nodes = [2, 4]
    ip_range = "10.1.0.0/16"
    name = "network12346"
    description = "newer network"
}

resource "grid_kubernetes" "k8s1" {
  network_name = grid_network.net1.name
  nodes_ip_range = grid_network.net1.nodes_ip_range 
  token = "12345678910122"
  ssh_key =  "${var.public_key}"

  master {
    disk_size = 22
    node = 2
    name = "mr"
    cpu = 2
    publicip = true
    memory = 2048
  }
  workers {
    disk_size = 15
    node = 2
    name = "w0"
    cpu = 2
    memory = 2048
  }
  workers {
    disk_size = 13
    node = 4
    name = "w2"
    cpu = 1 
    memory = 2048
  }
  workers {
    disk_size = 13
    node = 4
    name = "w3"
    cpu = 1
    memory = 2048
  }
}


output "master_public_ip" {
    value = grid_kubernetes.k8s1.master[0].computedip
}

output "wg_config" {
    value = grid_network.net1.access_wg_config
}
