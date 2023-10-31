terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

locals {
  servers_count = length(var.servers) + 1
  server_flist  = "https://hub.grid.tf/aelawady.3bot/abdulrahmanelawady-nomad-server-latest.flist"
  client_flist  = "https://hub.grid.tf/aelawady.3bot/abdulrahmanelawady-nomad-client-latest.flist"
  entrypoint    = "/sbin/zinit init"
}

resource "grid_network" "net" {
  nodes       = var.network.nodes
  ip_range    = var.network.ip_range
  name        = var.network.name
  description = var.network.description
}

resource "grid_deployment" "server1" {
  node         = var.servers[0].node
  network_name = grid_network.net.name

  vms {
    name       = var.servers[0].name
    flist      = local.server_flist
    cpu        = var.servers[0].cpu
    memory     = var.servers[0].memory
    ip         = var.first_server_ip
    publicip   = var.servers[0].publicip
    publicip6  = var.servers[0].publicip6
    planetary  = var.servers[0].planetary
    entrypoint = local.entrypoint
    mounts {
      disk_name   = var.servers[0].disk.name
      mount_point = var.servers[0].mount_point
    }
    env_vars = {
      SSH_KEY       = var.ssh_key
      NOMAD_SERVERS = local.servers_count
    }
  }

  disks {
    name = var.servers[0].disk.name
    size = var.servers[0].disk.size
  }
}

resource "grid_deployment" "servers" {
  for_each     = { for i, s in var.servers : s.node => s... if i != 0 }
  node         = tonumber(each.key)
  network_name = grid_network.net.name

  dynamic "vms" {
    for_each = each.value
    content {
      name       = vms.value.name
      flist      = local.server_flist
      cpu        = vms.value.cpu
      memory     = vms.value.memory
      publicip   = vms.value.publicip
      publicip6  = vms.value.publicip6
      planetary  = vms.value.planetary
      entrypoint = local.entrypoint
      mounts {
        disk_name   = vms.value.disk.name
        mount_point = vms.value.mount_point
      }
      env_vars = {
        SSH_KEY         = var.ssh_key
        FIRST_SERVER_IP = var.first_server_ip
        NOMAD_SERVERS   = local.servers_count
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
  node         = tonumber(each.key)
  network_name = grid_network.net.name

  dynamic "vms" {
    for_each = each.value
    content {
      name       = vms.value.name
      flist      = local.client_flist
      cpu        = vms.value.cpu
      memory     = vms.value.memory
      publicip   = vms.value.publicip
      publicip6  = vms.value.publicip6
      planetary  = vms.value.planetary
      entrypoint = local.entrypoint
      mounts {
        disk_name   = vms.value.disk.name
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
