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
  for_each     = { for i, s in var.servers : i => s }
  node         = each.value.node
  name         = each.value.name
  network_name = grid_network.net.name

  vms {
    name        = each.value.name
    flist       = local.server_flist
    cpu         = each.value.cpu
    memory      = each.value.memory
    rootfs_size = each.value.rootfs_size
    ip          = each.key == "0" ? var.first_server_ip : null
    publicip    = each.value.publicip
    publicip6   = each.value.publicip6
    planetary   = each.value.planetary
    entrypoint  = local.entrypoint
    mounts {
      disk_name   = each.value.disk.name
      mount_point = each.value.mount_point
    }
    env_vars = {
      SSH_KEY         = var.ssh_key
      NOMAD_SERVERS   = local.servers_count
      FIRST_SERVER_IP = each.key != "0" ? var.first_server_ip : null
    }
  }

  disks {
    name = each.value.disk.name
    size = each.value.disk.size
  }
}

resource "grid_deployment" "clients" {
  for_each     = { for i, c in var.clients : i => c }
  node         = each.value.node
  name         = each.value.name
  network_name = grid_network.net.name

  vms {
    name        = each.value.name
    flist       = local.client_flist
    cpu         = each.value.cpu
    memory      = each.value.memory
    rootfs_size = each.value.rootfs_size
    publicip    = each.value.publicip
    publicip6   = each.value.publicip6
    planetary   = each.value.planetary
    entrypoint  = local.entrypoint
    mounts {
      disk_name   = each.value.disk.name
      mount_point = each.value.mount_point
    }
    env_vars = {
      SSH_KEY         = var.ssh_key
      FIRST_SERVER_IP = var.first_server_ip
    }
  }

  disks {
    name = each.value.disk.name
    size = each.value.disk.size
  }
}
