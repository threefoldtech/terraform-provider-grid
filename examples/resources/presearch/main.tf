terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}
locals {
  solution_type = "Presearch"
  name          = "mypreasearch"
}


resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = [8]
  ip_range      = "10.1.0.0/16"
  description   = "newer network"
  add_wg_access = true
}

# Deployment specs
resource "grid_deployment" "d1" {
  solution_type = local.solution_type
  name          = local.name
  node          = 8
  network_name  = grid_network.net1.name

  disks {
    name        = "data"
    size        = 10
    description = "volume holding docker data"
  }

  vms {
    name       = "presearch"
    flist      = "https://hub.grid.tf/tf-official-apps/presearch-v2.2.flist"
    entrypoint = "/sbin/zinit init"
    publicip   = true
    planetary  = true
    cpu        = 1
    memory     = 1024

    mounts {
      disk_name   = "data"
      mount_point = "/var/lib/docker"
    }

    env_vars = {
      SSH_KEY                     = file("~/.ssh/id_rsa.pub"),
      PRESEARCH_REGISTRATION_CODE = "",

      # COMMENT the two env vars below to create a new node. 
      # or uncomment and fill them from your old node to restore it. #

      # optional keys pair from the old node 
      # important to follow the schema ` <<-EOF ... EOF ` with no indentation
      PRESEARCH_BACKUP_PRI_KEY = <<EOF
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDQjfuZ3uIGOXUP
Qqpw1K85LV6sZWOAntUnhL73GXTWcwBer06yPI1ush8Vj6tdP94hmUFfWW85vYRU
...
-----END PRIVATE KEY-----
      EOF
      PRESEARCH_BACKUP_PUB_KEY = <<EOF
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0I37md7iBjl1D0KqcNSv
OS1erGVjgJ7VJ4S+9xl01nMAXq9OsjyNbrIfFY+rXT/eIZlBX1lvOb2EVJ93o1mz
...
-----END PUBLIC KEY-----
      EOF
    }
  }
}


# Print deployment info
output "node1_zmachine1_ip" {
  value = grid_deployment.d1.vms[0].ip
}

output "computed_public_ip" {
  value = grid_deployment.d1.vms[0].computedip
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].ygg_ip
}
