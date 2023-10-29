terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

locals {
  names        = concat([for s in var.nomad.servers : s.name], [for c in var.nomad.clients : c.name])
  server_flist = "https://hub.grid.tf/aelawady.3bot/abdulrahmanelawady-nomad-server-latest.flist"
  client_flist = "https://hub.grid.tf/aelawady.3bot/abdulrahmanelawady-nomad-client-latest.flist"
  entrypoint   = "/sbin/zinit init"
}

resource "grid_network" "net" {
  nodes       = var.network.nodes
  ip_range    = var.network.ip_range
  name        = var.network.name
  description = var.network.description
}

resource "grid_deployment" "nomad" {
  name         = var.nomad.name
  node         = var.nomad.node
  network_name = grid_network.net.name

  vms {
    name       = var.nomad.servers[0].name
    flist      = local.server_flist
    cpu        = var.nomad.servers[0].cpu
    memory     = var.nomad.servers[0].memory
    entrypoint = local.entrypoint
    ip         = var.first_server_ip
    env_vars = {
      SSH_KEY = var.ssh_key
    }
    planetary = var.nomad.servers[0].planetary
  }

  vms {
    name       = var.nomad.servers[1].name
    flist      = local.server_flist
    cpu        = var.nomad.servers[1].cpu
    memory     = var.nomad.servers[1].memory
    entrypoint = local.entrypoint
    env_vars = {
      SSH_KEY         = var.ssh_key
      FIRST_SERVER_IP = var.first_server_ip
    }
    planetary = var.nomad.servers[1].planetary
  }

  vms {
    name       = var.nomad.servers[2].name
    flist      = local.server_flist
    cpu        = var.nomad.servers[2].cpu
    memory     = var.nomad.servers[2].memory
    entrypoint = local.entrypoint
    env_vars = {
      SSH_KEY         = var.ssh_key
      FIRST_SERVER_IP = var.first_server_ip
    }
    planetary = var.nomad.servers[2].planetary
  }

  dynamic "vms" {
    for_each = var.nomad.clients
    content {
      name       = vms.value.name
      flist      = local.client_flist
      cpu        = vms.value.cpu
      memory     = vms.value.memory
      entrypoint = local.entrypoint
      env_vars = {
        SSH_KEY         = var.ssh_key
        FIRST_SERVER_IP = var.first_server_ip
      }
      planetary = vms.value.planetary
    }
  }
}


