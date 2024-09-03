terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

locals {
  servers_count = length(var.servers)
  server_flist  = "https://hub.grid.tf/aelawady.3bot/abdulrahmanelawady-nomad-server-latest.flist"
  client_flist  = "https://hub.grid.tf/aelawady.3bot/abdulrahmanelawady-nomad-client-latest.flist"
  entrypoint    = "/sbin/zinit init"
}

resource "grid_network" "net" {
  name        = var.network.name
  nodes       = var.network.nodes
  ip_range    = var.network.ip_range
  description = var.network.description
}

resource "grid_deployment" "servers" {
  for_each     = { for s in var.servers : s.node => s... }
  node         = each.key
  network_name = grid_network.net.name

  dynamic "vms" {
    for_each = each.value
    content {
      name        = vms.value.name
      flist       = local.server_flist
      cpu         = vms.value.cpu
      ip          = vms.value == var.servers[0] ? var.first_server_ip : null
      memory      = vms.value.memory
      rootfs_size = vms.value.rootfs_size
      publicip    = vms.value.publicip
      publicip6   = vms.value.publicip6
      planetary   = vms.value.planetary
      entrypoint  = local.entrypoint
      mounts {
        name        = vms.value.disk.name
        mount_point = vms.value.mount_point
      }
      env_vars = {
        SSH_KEY         = var.ssh_key
        NOMAD_SERVERS   = local.servers_count
        FIRST_SERVER_IP = vms.value != var.servers[0] ? var.first_server_ip : null
      }
    }
  }

  dynamic "disks" {
    for_each = each.value
    content {
      name = disks.value.disk.name
      size = disks.value.disk.size
    }
  }
}

resource "grid_deployment" "clients" {
  for_each     = { for c in var.clients : c.node => c... }
  node         = each.key
  network_name = grid_network.net.name

  dynamic "vms" {
    for_each = each.value
    content {
      name        = vms.value.name
      flist       = local.client_flist
      cpu         = vms.value.cpu
      memory      = vms.value.memory
      rootfs_size = vms.value.rootfs_size
      publicip    = vms.value.publicip
      publicip6   = vms.value.publicip6
      planetary   = vms.value.planetary
      entrypoint  = local.entrypoint
      mounts {
        name        = vms.value.disk.name
        mount_point = vms.value.mount_point
      }
      env_vars = {
        SSH_KEY         = var.ssh_key
        FIRST_SERVER_IP = var.first_server_ip
      }
    }
  }

  dynamic "disks" {
    for_each = each.value
    content {
      name = disks.value.disk.name
      size = disks.value.disk.size
    }
  }
}
