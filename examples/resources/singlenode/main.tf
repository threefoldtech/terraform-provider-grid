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
    nodes = [2,3]
    ip_range = "10.1.0.0/16"
    name = "network"
    description = "newer network"
    add_wg_access = true
}
resource "grid_deployment" "d1" {
  node = 2
  network_name = grid_network.net1.name
  ip_range = lookup(grid_network.net1.nodes_ip_range, 2, "")
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 2 
    publicip = true
    memory = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCnJiB+sXPfqMKx6g67qOONqD0Kme08OKPtvacRwP6o0T8b404YaBvJaAEnsTgNE5A8vl15LBHdI94MdNoNwV7xT9HWWlw6hQ8PzN0e6z5M5bsKH6R6cKVbg8iYUESWWkRBr8iN10KDmbysyIbpr1QVIUoAbET4XpkJMxw46L8ClBsfPY5YFd1Bdd1oLwHJD4+cZQdZf9iSV4EtVfOfgpmOqk5mTzJGEnVf2/NnwzvTeuiezqY9QeIpigHvCKuj4JMyxLYk7zz6/5qY85v1yIUlMQ7xO3OWQFboNYr8E1O6w3wNGp3kGzbI8YrXankz3jfR2tFQBk7f4uWFzjYeaFv04QP830I0l/OSNrM4xBQ8JAQ20PxG2xznfY45g/gDTA2KxKEHLcpxZvq1aLTiqXOay0a270QMVIRIbK69Pov4y94TAZnDqf0DJpDo+dauH/TfDbtA/xelProl7CncE8ZG+HKrkYaNQef8YTql+9jLZwY9IMViwGrKJky6B5lzhQc= khaled@khaled-Inspiron-3576"
    }
    planetary = true
  }
  vms {
    name = "anothervm"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 1
    memory = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCnJiB+sXPfqMKx6g67qOONqD0Kme08OKPtvacRwP6o0T8b404YaBvJaAEnsTgNE5A8vl15LBHdI94MdNoNwV7xT9HWWlw6hQ8PzN0e6z5M5bsKH6R6cKVbg8iYUESWWkRBr8iN10KDmbysyIbpr1QVIUoAbET4XpkJMxw46L8ClBsfPY5YFd1Bdd1oLwHJD4+cZQdZf9iSV4EtVfOfgpmOqk5mTzJGEnVf2/NnwzvTeuiezqY9QeIpigHvCKuj4JMyxLYk7zz6/5qY85v1yIUlMQ7xO3OWQFboNYr8E1O6w3wNGp3kGzbI8YrXankz3jfR2tFQBk7f4uWFzjYeaFv04QP830I0l/OSNrM4xBQ8JAQ20PxG2xznfY45g/gDTA2KxKEHLcpxZvq1aLTiqXOay0a270QMVIRIbK69Pov4y94TAZnDqf0DJpDo+dauH/TfDbtA/xelProl7CncE8ZG+HKrkYaNQef8YTql+9jLZwY9IMViwGrKJky6B5lzhQc= khaled@khaled-Inspiron-3576"
    }
  }
}
resource "grid_deployment" "d2" {
  node = 3
  network_name = grid_network.net1.name
  ip_range = lookup(grid_network.net1.nodes_ip_range, 3, "")
  vms {
    name = "vm222"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 2 
    publicip = true
    memory = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCnJiB+sXPfqMKx6g67qOONqD0Kme08OKPtvacRwP6o0T8b404YaBvJaAEnsTgNE5A8vl15LBHdI94MdNoNwV7xT9HWWlw6hQ8PzN0e6z5M5bsKH6R6cKVbg8iYUESWWkRBr8iN10KDmbysyIbpr1QVIUoAbET4XpkJMxw46L8ClBsfPY5YFd1Bdd1oLwHJD4+cZQdZf9iSV4EtVfOfgpmOqk5mTzJGEnVf2/NnwzvTeuiezqY9QeIpigHvCKuj4JMyxLYk7zz6/5qY85v1yIUlMQ7xO3OWQFboNYr8E1O6w3wNGp3kGzbI8YrXankz3jfR2tFQBk7f4uWFzjYeaFv04QP830I0l/OSNrM4xBQ8JAQ20PxG2xznfY45g/gDTA2KxKEHLcpxZvq1aLTiqXOay0a270QMVIRIbK69Pov4y94TAZnDqf0DJpDo+dauH/TfDbtA/xelProl7CncE8ZG+HKrkYaNQef8YTql+9jLZwY9IMViwGrKJky6B5lzhQc= khaled@khaled-Inspiron-3576"
    }
    planetary = true
  }
}
output "wg_config" {
    value = grid_network.net1.access_wg_config
}
output "node1_zmachine1_ip" {
    value = grid_deployment.d1.vms[0].ip
}
output "node1_zmachine2_ip" {
    value = grid_deployment.d1.vms[1].ip
}
output "public_ip" {
    value = grid_deployment.d1.vms[0].computedip
}

output "ygg_ip" {
    value = grid_deployment.d1.vms[0].ygg_ip
}
output "ygg_ip1" {
    value = grid_deployment.d2.vms[0].ygg_ip
}