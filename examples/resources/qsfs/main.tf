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
  datas = ["data1", "data2", "data3", "data4"]
}

resource "grid_network" "net1" {
  nodes       = [7]
  ip_range    = "10.1.0.0/16"
  name        = "network"
  description = "newer network"
}

resource "grid_deployment" "d1" {
  node = 7
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
  node         = 7
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
    name       = "vm"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 2
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    planetary  = true
    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCtCuUUCZGLZ4NoihAiUK8K0kSoTR1WgIaLQKqMdQ/99eocMLqJgQMRIp8lueFG7SpcgXVRzln8KNKZX1Hm8lcrXICr3dnTW/0bpEnF4QOGLYZ/qTLF5WmoCgKyJ6WO96GjWJBsZPads+RD0WeiijV7jj29lALsMAI8CuOH0pcYUwWsRX/I1z2goMPNRY+PBjknMYFXEqizfUXqUnpzF3w/bKe8f3gcrmOm/Dxh1nHceJDW52TJL/sPcl6oWnHZ3fY4meTiAS5NZglyBF5oKD463GJnMt/rQ1gDNl8E4jSJUArN7GBJntTYxFoFo6zxB1OsSPr/7zLfPG420+9saBu9yN1O9DlSwn1ZX+Jg0k7VFbUpKObaCKRmkKfLiXJdxkKFH/+qBoCCnM5hfYxAKAyQ3YCCP/j9wJMBkbvE1QJMuuoeptNIvSQW6WgwBfKIK0shsmhK2TDdk0AHEnzxPSkVGV92jygSLeZ4ur/MZqWDx/b+gACj65M3Y7tzSpsR76M= omar@omar-Predator-PT315-52"
    }
    mounts {
      disk_name   = "qsfs"
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
