terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}


resource "grid_scheduler" "sched" {
  requests {
    name = "node1"
    cru  = 2
    sru  = 1024 * 5
    mru  = 1024 * 2
  }
  requests {
    name = "node2"
    cru  = 2
    sru  = 1024 * 4
    mru  = 1024 * 2
  }
}

resource "grid_network" "net1" {
  nodes       = [grid_scheduler.sched.nodes["node1"], grid_scheduler.sched.nodes["node2"]]
  ip_range    = "172.20.0.0/16"
  name        = "net1"
  description = "new network"
}

resource "grid_deployment" "d1" {
  node         = grid_scheduler.sched.nodes["node1"]
  network_name = grid_network.net1.name
  zdbs {
    name        = "zdb1"
    size        = 1
    description = "zdb1 description"
    password    = "zdbpasswd1"
    mode        = "user"
  }
  zdbs {
    name        = "zdb2"
    size        = 2
    description = "zdb2 description"
    password    = "zdbpasswd2"
    mode        = "seq"
  }
  disks {
    name        = "mydisk1"
    size        = 2
    description = "this is my disk description2323saqa"

  }
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 1
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      key   = "SSH_KEY"
      value = file("~/.ssh/id_rsa.pub")
    }

  }

  vms {
    name       = "vm2"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 1
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      key   = "SSH_KEY"
      value = file("~/.ssh/id_rsa.pub")
    }
  }
}

resource "grid_deployment" "d2" {
  node         = grid_scheduler.sched.nodes["node2"]
  network_name = grid_network.net1.name
  disks {
    name        = "mydisk1"
    size        = 2
    description = "this is my disk description2323saqs"
  }
  disks {
    name        = "mydisk2"
    size        = 2
    description = "this is my disk2"
  }
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 1
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    mounts {
      disk_name   = "mydisk1"
      mount_point = "/opt"
    }
    mounts {
      disk_name   = "mydisk2"
      mount_point = "/test"
    }
    env_vars = {
      key   = "SSH_KEY"
      value = file("~/.ssh/id_rsa.pub")
    }
  }
  vms {
    name       = "vm2"
    flist      = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu        = 1
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    mounts {
      disk_name   = "mydisk2"
      mount_point = "/test"
    }
    env_vars = {
      key   = "SSH_KEY"
      value = file("~/.ssh/id_rsa.pub")
    }
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

