terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}

locals {
  metas = ["meta1", "meta2", "meta3", "meta4"]
  datas = ["data1", "data2", "data3", "data4"]
}

resource "grid_scheduler" "sched" {
  requests {
    name      = "node"
    cru       = 2
    sru       = 10 * 1024
    mru       = 1024
    hru       = 8 * 10 * 1024
    yggdrasil = true
    wireguard = false
  }
}

resource "grid_network" "net1" {
  name        = "network"
  nodes       = [grid_scheduler.sched.nodes["node"]]
  ip_range    = "10.1.0.0/16"
  description = "qsfs network"
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["node"]) = random_bytes.mycelium_key.hex
  }
}

resource "grid_deployment" "d1" {
  node = grid_scheduler.sched.nodes["node"]
  dynamic "zdbs" {
    for_each = local.metas
    content {
      name        = zdbs.value
      description = "description"
      password    = "password"
      size        = 10
      mode        = "user"
    }
  }
  dynamic "zdbs" {
    for_each = local.datas
    content {
      name        = zdbs.value
      description = "description"
      password    = "password"
      size        = 10
      mode        = "seq"
    }
  }
}

resource "grid_deployment" "qsfs" {
  node         = grid_scheduler.sched.nodes["node"]
  network_name = grid_network.net1.name
  qsfs {
    name                  = "qsfs"
    description           = "description6"
    cache                 = 10240 # 10 GB
    minimal_shards        = 2
    expected_shards       = 4
    redundant_groups      = 0
    redundant_nodes       = 0
    max_zdb_data_dir_size = 512 # 512 MB
    encryption_algorithm  = "AES"
    encryption_key        = "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af"
    compression_algorithm = "snappy"
    metadata {
      type                 = "zdb"
      prefix               = "hamada"
      encryption_algorithm = "AES"
      encryption_key       = "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af"
      dynamic "backends" {
        for_each = [for zdb in grid_deployment.d1.zdbs : zdb if zdb.mode != "seq"]
        content {
          address   = format("[%s]:%d", backends.value.ips[1], backends.value.port)
          namespace = backends.value.namespace
          password  = backends.value.password
        }
      }
    }
    groups {
      dynamic "backends" {
        for_each = [for zdb in grid_deployment.d1.zdbs : zdb if zdb.mode == "seq"]
        content {
          address   = format("[%s]:%d", backends.value.ips[1], backends.value.port)
          namespace = backends.value.namespace
          password  = backends.value.password
        }
      }
    }
  }
  vms {
    name             = "vm"
    flist            = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu              = 2
    memory           = 1024
    entrypoint       = "/sbin/zinit init"
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
    mounts {
      name        = "qsfs"
      mount_point = "/qsfs"
    }
  }
}

output "metrics" {
  value = grid_deployment.qsfs.qsfs[0].metrics_endpoint
}

output "mycelium_ip" {
  value = grid_deployment.qsfs.vms[0].mycelium_ip
}
