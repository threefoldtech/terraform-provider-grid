module "kubernetes" {
  source  = "github.com/threefoldtech/terraform-provider-grid/modules/k8s-module"
  token   = local.token
  network = local.network
  master  = var.master
  ssh     = var.ssh
  workers = var.workers
  disks   = var.disks
}

variable "ssh" {
  type = string
}

variable "network_nodes" {
  type = list(number)
}

variable "master" {
  type = object({
    name        = string
    node        = number
    cpu         = number
    memory      = number
    disk_name   = string
    mount_point = string
    publicip    = bool
    planetary   = bool
  })
}

variable "workers" {
  type = list(object({
    name        = string
    node        = number
    cpu         = number
    memory      = number
    disk_name   = string
    mount_point = string
    publicip    = bool
    planetary   = bool
  }))
}

variable "disks" {
  type = list(object({
    node        = number
    name        = string
    size        = number
    description = string
  }))
}

locals {
  token = "838a6db4"

  network = {
    nodes         = var.network_nodes
    ip_range      = "10.1.0.0/16"
    name          = "test_network"
    description   = "new network for testing"
    add_wg_access = true
  }
}

output "master_yggip" {
  value = module.kubernetes.master.ygg_ip
}
