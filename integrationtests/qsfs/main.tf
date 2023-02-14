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

locals {
  metas = ["meta1", "meta2", "meta3", "meta4"]
  datas = ["data1", "data2", "data3", "data4", "data5", "data6", "data7", "data8"]
}

resource "grid_network" "net1" {
    nodes = [27]
    ip_range = "10.1.0.0/16"
    name = "net1"
    description = "newer network"
}

resource "grid_deployment" "d1" {
    node_id= 27
    dynamic "zdbs" {
        for_each = local.metas
        content {
            name = zdbs.value
            description = "description"
            password = "password"
            size = 1
            mode = "user"
        }
    }
    dynamic "zdbs" {
        for_each = local.datas
        content {
            name = zdbs.value
            description = "description"
            password = "password"
            size = 1
            mode = "seq"
        }
    }
}

resource "grid_deployment" "qsfs" {
  node_id= 27
  network_name = grid_network.net1.name
  qsfs {
    name = "qsfs"
    description = "description6"
    cache = 2048 
    minimal_shards = 2
    expected_shards = 3
    redundant_groups = 0
    redundant_nodes = 0
    max_zdb_data_dir_size = 2048 # 2 GB
    encryption_algorithm = "AES"
    encryption_key = "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af"
    compression_algorithm = "snappy"
    metadata {
      type = "zdb"
      prefix = "hamada"
      encryption_algorithm = "AES"
      encryption_key = "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af"
      dynamic "backends" {
          for_each = [for zdb in grid_deployment.d1.zdbs : zdb if zdb.mode != "seq"]
          content {
              address = format("[%s]:%d", backends.value.ips[1], backends.value.port)
              namespace = backends.value.namespace
              password = backends.value.password
          }
      }
    }
    groups {
      dynamic "backends" {
          for_each = [for zdb in grid_deployment.d1.zdbs : zdb if zdb.mode == "seq"]
          content {
              address = format("[%s]:%d", backends.value.ips[1], backends.value.port)
              namespace = backends.value.namespace
              password = backends.value.password
          }
      }
    }
  }
  vms {
    name = "vm"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 2
    memory = 1024
    entrypoint = "/sbin/zinit init"
    planetary = true
    env_vars = {
      SSH_KEY = "${var.public_key}"
    }
    mounts {
        disk_name = "qsfs"
        mount_point = "/qsfs"
    }
  }
}
output "metrics" {
    value = grid_deployment.qsfs.qsfs[0].metrics_endpoint
}
output "ygg_ip" {
    value = grid_deployment.qsfs.vms[0].ygg_ip
}
