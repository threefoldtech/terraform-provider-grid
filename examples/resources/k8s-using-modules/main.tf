module "kubernetes" {
  source  = "github.com/threefoldtech/terraform-provider-grid/modules/k8s-module"
  ssh     = local.ssh
  token   = local.token
  network = local.network
  master  = local.master
  workers = local.workers
  disks   = local.disks
}

locals {
  ssh   = file("~/.ssh/id_rsa.pub")
  token = "838a6db4"

  network = {
    nodes         = [45]
    ip_range      = "10.1.0.0/16"
    name          = "test_network"
    description   = "new network for testing"
    add_wg_access = false
  }

  master = {
    name        = "mr"
    node        = 45
    cpu         = 2
    memory      = 1024
    disk_name   = "mrdisk"
    mount_point = "/mydisk"
    publicip    = true
    planetary   = false
  }

  workers = [
    {
      name        = "w0"
      node        = 45
      cpu         = 1
      memory      = 1024
      disk_name   = "w0disk"
      mount_point = "/mydisk"
      publicip    = false
      planetary   = false
    },
  ]

  disks = [
    {
      name        = "mrdisk"
      node        = 45
      size        = 5
      description = ""
    },
    {
      name        = "w0disk"
      node        = 45
      size        = 2
      description = ""
    },
  ]
}
