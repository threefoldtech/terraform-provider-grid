terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

locals {
  names       = concat([for w in var.workers : w.name], [var.master.name])
  disks_map   = { for d in var.disks : d.node => d... }
  master_disk = lookup(local.disks_map, "${var.master.node}", {})[0]
  vms_list    = flatten([for node in grid_deployment.workers : node.vms])
}

module "validator" {
  source = "./validator"
  names  = local.names
}

resource "grid_network" "net" {
  nodes         = var.network.nodes
  ip_range      = var.network.ip_range
  name          = var.network.name
  description   = var.network.description
  add_wg_access = var.network.add_wg_access
}

resource "grid_deployment" "master" {
  node         = var.master.node
  network_name = grid_network.net.name
  vms {
    name       = var.master.name
    flist      = "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist"
    cpu        = var.master.cpu
    publicip   = var.master.publicip
    planetary  = var.master.planetary
    memory     = var.master.memory
    entrypoint = "/sbin/zinit init"
    mounts {
      disk_name   = var.master.disk_name
      mount_point = var.master.mount_point
    }
    env_vars = {
      SSH_KEY           = "${var.ssh}"
      K3S_TOKEN         = "${var.token}"
      K3S_DATA_DIR      = "/mydisk"
      K3S_FLANNEL_IFACE = "eth0"
      K3S_NODE_NAME     = "${var.master.name}"
      K3S_URL           = ""
    }
  }

  disks {
    name        = local.master_disk.name
    size        = local.master_disk.size
    description = local.master_disk.description
  }
}

resource "grid_deployment" "workers" {
  for_each     = { for w in var.workers : w.node => w... }
  node         = tonumber(each.key)
  network_name = grid_network.net.name

  dynamic "vms" {
    for_each = each.value
    content {
      name       = vms.value.name
      flist      = "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist"
      cpu        = vms.value.cpu
      memory     = vms.value.memory
      publicip   = vms.value.publicip
      planetary  = vms.value.planetary
      entrypoint = "/sbin/zinit init"
      env_vars = {
        SSH_KEY           = "${var.ssh}"
        K3S_TOKEN         = "${var.token}"
        K3S_DATA_DIR      = "/mydisk"
        K3S_FLANNEL_IFACE = "eth0"
        K3S_NODE_NAME     = "${vms.value.name}"
        K3S_URL           = "https://${grid_deployment.master.vms[0].ip}:6443"
      }
      mounts {
        disk_name   = vms.value.disk_name
        mount_point = vms.value.mount_point
      }
    }
  }

  dynamic "disks" {
    for_each = lookup(local.disks_map, each.key, {})
    content {
      name        = disks.value.name
      size        = disks.value.size
      description = disks.value.description
    }
  }
}
