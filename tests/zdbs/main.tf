terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "grid_deployment" "d1" {
  node = 3
  
  zdbs{
    name = "zdb1"
    size = 10 
    description = "zdb1 description"
    password = "zdbpasswd1"
    mode = "user"
  }
}

resource "grid_network" "net1" {
    nodes = [2, 4]
    ip_range = "10.1.0.0/16"
    name = "network"
    description = "newer network"
    add_wg_access = true
}
resource "grid_deployment" "d11" {
  node = 2
  network_name = grid_network.net1.name
  ip_range = grid_network.net1.nodes_ip_range["2"]
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 1
    publicip = false
    memory = 1024
    entrypoint = "/sbin/zinit init"
    env_vars {
      key = "SSH_KEY"
      value = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDVaulbZarLfnE/9flAtT85cezs46qDdgaCHdliQRST16WrJ6jukZvTpWjUo0K/v+UHmxCnkY4KLv3Yl1cWSUFdlSTe21hoK7xEyU9F21PbhzKQASQga/J4QUxyZpdMybzt9ndlFsOy7x0xNUSMljf38za2yHGcpKGRw1Ungd4PBWKv0sKTKYjhHoHCTa72NAc95EMFJpvfowVWJvsxgjcHjOSaQpRQiXb3eHfMj5I4h7yhJOFin7GVev6bSwPEJRN+ydGiCmv3paNGvJbFEOROIBAp6q78RDf4rNyf3Vr244yB8ffl6bKpPb4LA0Ntex+e7jaxzBWAboiWXxDEadG2P/RAfciXasICZXLcZTfjlbXJ1OHTe/aStEgy36IWvt/SwKcazG/2V0enc3UwE/SzeqGyPT9A8HhuMn9TB4cYOTh9146Gl63EtTBnFX3z5EqQwytFyyxXToqDOuJAMYB2gKHsB1ePhWZOLxiguOPlU3ZSHm9XI+wZEcBNJ6B2yOM= hassan@hassan-Inspiron-3576"
    }
  }
  
}
output "wg_config" {
    value = grid_network.net1.access_wg_config
}
output "node1_container1_ip" {
    value = grid_deployment.d11.vms[0].ip
}

output "deployment_id" {
    value = grid_deployment.d1.id
}

output "zdb1_endpoint" {
    value = format("[%s]:%d", grid_deployment.d1.zdbs[0].ips[0], grid_deployment.d1.zdbs[0].port)
}

output "zdb1_namespace" {
    value = grid_deployment.d1.zdbs[0].namespace
}
