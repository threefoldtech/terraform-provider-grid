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
    nodes = [5]
    ip_range = "10.1.0.0/16"
    name = "network"
    description = "newer network"
}

resource "grid_deployment" "d1" {
    node = 5
    zdbs {
        name = "meta1"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "user"
    }
    zdbs {
        name = "meta2"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "user"
    }
    zdbs {
        name = "meta3"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "user"
    }
    zdbs {
        name = "meta4"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "user"
    }
    zdbs {
        name = "data1"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "seq"
    }
    zdbs {
        name = "data2"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "seq"
    }
    zdbs {
        name = "data3"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "seq"
    }
    zdbs {
        name = "data4"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "seq"
    }
    zdbs {
        name = "data5"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "seq"
    }
    zdbs {
        name = "data6"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "seq"
    }
    zdbs {
        name = "data7"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "seq"
    }
    zdbs {
        name = "data8"
        size = 10
        description = "zdb1 description"
        password = "password"
        mode = "seq"
    }
}

resource "grid_deployment" "qsfs" {
  node = 5
  network_name = grid_network.net1.name
  ip_range = grid_network.net1.nodes_ip_range[5]
  qsfs {
    name = "qsfs"
    description = "description2"
    cache = 1
    minimal_shards = 2
    expected_shards = 3
    redundant_groups = 0
    redundant_nodes = 0
    max_zdb_data_dir_size = 256000000
    encryption_algorithm = "AES"
    encryption_key = "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af"
    compression_algorithm = "snappy"
    metadata {
      type = "zdb"
      prefix = "hamada"
      encryption_algorithm = "AES"
      encryption_key = "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af"
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[0].ips[1], grid_deployment.d1.zdbs[0].port)
        namespace = grid_deployment.d1.zdbs[0].namespace
        password = grid_deployment.d1.zdbs[0].password
      }
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[1].ips[1], grid_deployment.d1.zdbs[1].port)
        namespace = grid_deployment.d1.zdbs[1].namespace
        password = grid_deployment.d1.zdbs[1].password
      }
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[2].ips[1], grid_deployment.d1.zdbs[2].port)
        namespace = grid_deployment.d1.zdbs[2].namespace
        password = grid_deployment.d1.zdbs[2].password
      }
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[3].ips[1], grid_deployment.d1.zdbs[3].port)
        namespace = grid_deployment.d1.zdbs[3].namespace
        password = grid_deployment.d1.zdbs[3].password
      }
    }
    groups {
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[4].ips[1], grid_deployment.d1.zdbs[4].port)
        namespace = grid_deployment.d1.zdbs[4].namespace
        password = grid_deployment.d1.zdbs[4].password
      }
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[5].ips[1], grid_deployment.d1.zdbs[5].port)
        namespace = grid_deployment.d1.zdbs[5].namespace
        password = grid_deployment.d1.zdbs[5].password
      }
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[6].ips[1], grid_deployment.d1.zdbs[6].port)
        namespace = grid_deployment.d1.zdbs[6].namespace
        password = grid_deployment.d1.zdbs[6].password
      }
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[7].ips[1], grid_deployment.d1.zdbs[7].port)
        namespace = grid_deployment.d1.zdbs[7].namespace
        password = grid_deployment.d1.zdbs[7].password
      }
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[8].ips[1], grid_deployment.d1.zdbs[8].port)
        namespace = grid_deployment.d1.zdbs[8].namespace
        password = grid_deployment.d1.zdbs[8].password
      }
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[9].ips[1], grid_deployment.d1.zdbs[9].port)
        namespace = grid_deployment.d1.zdbs[9].namespace
        password = grid_deployment.d1.zdbs[9].password
      }
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[10].ips[1], grid_deployment.d1.zdbs[10].port)
        namespace = grid_deployment.d1.zdbs[10].namespace
        password = grid_deployment.d1.zdbs[10].password
      }
      backends {
        address = format("[%s]:%d", grid_deployment.d1.zdbs[11].ips[1], grid_deployment.d1.zdbs[11].port)
        namespace = grid_deployment.d1.zdbs[11].namespace
        password = grid_deployment.d1.zdbs[11].password
      }
    }
  }
  vms {
    name = "vm"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 1
    description = "omar"
    memory = 1024
    entrypoint = "/sbin/zinit init"
    planetary = true
    env_vars {
      key = "SSH_KEY"
      value = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCtCuUUCZGLZ4NoihAiUK8K0kSoTR1WgIaLQKqMdQ/99eocMLqJgQMRIp8lueFG7SpcgXVRzln8KNKZX1Hm8lcrXICr3dnTW/0bpEnF4QOGLYZ/qTLF5WmoCgKyJ6WO96GjWJBsZPads+RD0WeiijV7jj29lALsMAI8CuOH0pcYUwWsRX/I1z2goMPNRY+PBjknMYFXEqizfUXqUnpzF3w/bKe8f3gcrmOm/Dxh1nHceJDW52TJL/sPcl6oWnHZ3fY4meTiAS5NZglyBF5oKD463GJnMt/rQ1gDNl8E4jSJUArN7GBJntTYxFoFo6zxB1OsSPr/7zLfPG420+9saBu9yN1O9DlSwn1ZX+Jg0k7VFbUpKObaCKRmkKfLiXJdxkKFH/+qBoCCnM5hfYxAKAyQ3YCCP/j9wJMBkbvE1QJMuuoeptNIvSQW6WgwBfKIK0shsmhK2TDdk0AHEnzxPSkVGV92jygSLeZ4ur/MZqWDx/b+gACj65M3Y7tzSpsR76M= omar@omar-Predator-PT315-52"
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