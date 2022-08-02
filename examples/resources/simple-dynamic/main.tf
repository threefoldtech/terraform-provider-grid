terraform {
  required_providers {
    grid = {
      source = "threefoldtechdev.com/providers/grid"
    }
  }
}

provider "grid" {
}

resource "grid_scheduler" "sched" {
  # a machine for the first server instance
  requests {
    name = "server1"
    cru = 1
    sru = 256
    mru = 256
  }
  # a machine for the second server instance
  requests {
    name = "server2"
    cru = 1
    sru = 256
    mru = 256
  }
  # a name workload
  requests {
    name = "gateway"
    ipv4 = true
    farm = "Freefarm"
  }
}

resource "grid_network" "net1" {
    nodes = distinct([
      grid_scheduler.sched.nodes["server1"],
      grid_scheduler.sched.nodes["server2"]
    ])
    ip_range = "10.1.0.0/16"
    name = "network2"
    description = "newer network"
}

resource "grid_deployment" "server1" {
  node = grid_scheduler.sched.nodes["server1"]
  network_name = grid_network.net1.name
  ip_range = lookup(grid_network.net1.nodes_ip_range, grid_scheduler.sched.nodes["server1"], "")
  vms {
    name = "firstserver"
    flist = "https://hub.grid.tf/omar0.3bot/omarelawady-simple-http-server-latest.flist"
    cpu = 1
    memory = 256
    rootfs_size = 256
    entrypoint = "/main.sh"
    env_vars {
      key = "SSH_KEY"
      value = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCtCuUUCZGLZ4NoihAiUK8K0kSoTR1WgIaLQKqMdQ/99eocMLqJgQMRIp8lueFG7SpcgXVRzln8KNKZX1Hm8lcrXICr3dnTW/0bpEnF4QOGLYZ/qTLF5WmoCgKyJ6WO96GjWJBsZPads+RD0WeiijV7jj29lALsMAI8CuOH0pcYUwWsRX/I1z2goMPNRY+PBjknMYFXEqizfUXqUnpzF3w/bKe8f3gcrmOm/Dxh1nHceJDW52TJL/sPcl6oWnHZ3fY4meTiAS5NZglyBF5oKD463GJnMt/rQ1gDNl8E4jSJUArN7GBJntTYxFoFo6zxB1OsSPr/7zLfPG420+9saBu9yN1O9DlSwn1ZX+Jg0k7VFbUpKObaCKRmkKfLiXJdxkKFH/+qBoCCnM5hfYxAKAyQ3YCCP/j9wJMBkbvE1QJMuuoeptNIvSQW6WgwBfKIK0shsmhK2TDdk0AHEnzxPSkVGV92jygSLeZ4ur/MZqWDx/b+gACj65M3Y7tzSpsR76M= omar@omar-Predator-PT315-52"
    }
    env_vars {
        key = "PATH"
        value = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    }

    planetary = true
  }
}

resource "grid_deployment" "server2" {
  node = grid_scheduler.sched.nodes["server2"]
  network_name = grid_network.net1.name
  ip_range = lookup(grid_network.net1.nodes_ip_range, grid_scheduler.sched.nodes["server2"], "")
  vms {
    name = "secondserver"
    flist = "https://hub.grid.tf/tf-official-apps/simple-http-server-latest.flist"
    cpu = 1
    memory = 256
    rootfs_size = 256
    entrypoint = "/main.sh"
    env_vars {
      key = "SSH_KEY"
      value = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCtCuUUCZGLZ4NoihAiUK8K0kSoTR1WgIaLQKqMdQ/99eocMLqJgQMRIp8lueFG7SpcgXVRzln8KNKZX1Hm8lcrXICr3dnTW/0bpEnF4QOGLYZ/qTLF5WmoCgKyJ6WO96GjWJBsZPads+RD0WeiijV7jj29lALsMAI8CuOH0pcYUwWsRX/I1z2goMPNRY+PBjknMYFXEqizfUXqUnpzF3w/bKe8f3gcrmOm/Dxh1nHceJDW52TJL/sPcl6oWnHZ3fY4meTiAS5NZglyBF5oKD463GJnMt/rQ1gDNl8E4jSJUArN7GBJntTYxFoFo6zxB1OsSPr/7zLfPG420+9saBu9yN1O9DlSwn1ZX+Jg0k7VFbUpKObaCKRmkKfLiXJdxkKFH/+qBoCCnM5hfYxAKAyQ3YCCP/j9wJMBkbvE1QJMuuoeptNIvSQW6WgwBfKIK0shsmhK2TDdk0AHEnzxPSkVGV92jygSLeZ4ur/MZqWDx/b+gACj65M3Y7tzSpsR76M= omar@omar-Predator-PT315-52"
    }
    env_vars {
        key = "PATH"
        value = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    }

    planetary = true
  }
}

resource "grid_fqdn_proxy" "balancer" {
  node = grid_scheduler.sched.nodes["gateway"]
  name = "balancer"
  fqdn = "remote.omar.grid.tf"
  backends = [format("http://[%s]", grid_deployment.server1.vms[0].ygg_ip), format("http://[%s]", grid_deployment.server2.vms[0].ygg_ip)]
  tls_passthrough = false
}


output "load_balancer_domain" {
    value = grid_fqdn_proxy.balancer.fqdn
}