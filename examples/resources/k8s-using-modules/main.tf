module "kubernetes" {
  source  = "github.com/threefoldtech/terraform-provider-grid/modules/k8s-module"
  ssh     = local.ssh
  token   = local.token
  network = local.network
  master  = local.master
  workers = local.workers
  disks   = local.disks
}
resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}
locals {
  ssh     = file("~/.ssh/id_rsa.pub")
  token   = "838a6db4"
  node_id = 11

  network = {
    nodes         = [local.node_id]
    ip_range      = "10.1.0.0/16"
    name          = "test_network"
    description   = "new network for testing"
    add_wg_access = false
    mycelium_keys = {
      format("%s", local.node_id) = random_bytes.mycelium_key.hex
    }
  }

  master = {
    name             = "mr"
    node             = local.node_id
    cpu              = 2
    memory           = 1024
    disk_name        = "mrdisk"
    mount_point      = "/mydisk"
    publicip         = true
    planetary        = false
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
  }

  workers = [
    {
      name             = "w0"
      node             = local.node_id
      cpu              = 1
      memory           = 1024
      disk_name        = "w0disk"
      mount_point      = "/mydisk"
      publicip         = false
      planetary        = false
      mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
    },
  ]

  disks = [
    {
      name        = "mrdisk"
      node        = local.node_id
      size        = 5
      description = ""
    },
    {
      name        = "w0disk"
      node        = local.node_id
      size        = 2
      description = ""
    },
  ]
}
output "master_mycelium_ip" {
  value = module.kubernetes.master.mycelium_ip
}

output "workers" {
  value = module.kubernetes.workers.w0.mycelium_ip
}
