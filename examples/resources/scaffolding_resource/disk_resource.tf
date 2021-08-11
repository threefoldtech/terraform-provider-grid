terraform {
  required_providers {
    grid = {
      version = "0.2"
      source  = "ashraffouda.com/edu/grid"
    }
  }
}

provider "grid" {}


resource "grid_deployment" "d1" {
  node = 2
  disks {
    name = "mydisk1"
    size = 2
    description = "this is my disk description2"
    
  }
  disks {
    name = "mydisk2"
    size=5
    description = "this is my disk2"
  }
  vms {
    name = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu = 1
    memory = 1024
    entrypoint = "/sbin/zinit init"
    mounts {
      disk_name="mydisk1"
      mount_point="/opt"
    }
    mounts {
      disk_name="mydisk2"
      mount_point = "/test"
    }

  }
}

