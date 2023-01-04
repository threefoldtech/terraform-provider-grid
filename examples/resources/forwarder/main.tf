terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}
provider "grid" {
  mnemonics = "winner giant reward damage expose pulse recipe manual brand volcano dry avoid"
  network   = "dev"
}

locals {
  name  = "myvm"
  name2 = "myvm2"
  node  = 34
  node2 = 49
}

resource "grid_network" "net1" {
  name        = local.name
  nodes       = [local.node]
  ip_range    = "10.1.0.0/16"
  description = "newer network"
  # add_wg_access = true
}
resource "grid_deployment" "d1" {
  name         = local.name
  node         = local.node
  network_name = grid_network.net1.name
  vms {
    name  = "vm1"
    flist = "https://hub.grid.tf/tf-official-apps/base:latest.flist"
    cpu   = 2
    # publicip   = true
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDNFMdYHGcGqWsE7H1eqsWaXwOQQQrh6bYWsKKGa7KswNa8BhyEK9bjxEs13LvIVPUckn/wVVqlH0qFAc8JjBRmSjGdDjyZIvawOIyDX/Jr0fPAyS3e8eL+FvuJVW1OCKZ4DmGYgNiEYFDZ0uxf6lyfJyYsiTxzeukHOjtDe3xIg660aYdWKV4bbog9AmkdXL7x0lTkUb+ERVhMCvtIFE7YKGZqeEovL6tgXl9U/ApdXK/xT0283CWoKBQVcvZUEqimtWTaEFekFD4PTDkwfUg6WZY6Gy6yTU4HESziSh5e0raH7mP4YJ8tZsdtnfIL+NRvReUqFz8goG6Dm0nvsvwcI8jJhH8lGbPxd6hqbvk+PnttZRr5uxiIJwIx/98fW+mAL0N7AScRklFSjQgr4dRTqZ+/TXyUj9E0x/nyaEpRuj83SzLSwFsc2izoxNCSJDz3m5t7RW2Inm3X3oZmkFOdWL4Y1yGIHcFY0i9LSgHYaQpfLpDz4WnlkkU8cyf73Ic= rawda@rawda-Inspiron-3576"
    }
    planetary = true
  }
}

output "vm1_private_ip" {
  value = grid_deployment.d1.vms[0].ip
}

resource "grid_deployment" "d2" {
  name         = local.name2
  node         = local.node2
  network_name = grid_network.net1.name
  vms {
    name       = "vm2"
    flist      = "https://hub.grid.tf/azmy.3bot/forwarder.flist"
    cpu        = 2
    publicip   = true
    memory     = 1024
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDNFMdYHGcGqWsE7H1eqsWaXwOQQQrh6bYWsKKGa7KswNa8BhyEK9bjxEs13LvIVPUckn/wVVqlH0qFAc8JjBRmSjGdDjyZIvawOIyDX/Jr0fPAyS3e8eL+FvuJVW1OCKZ4DmGYgNiEYFDZ0uxf6lyfJyYsiTxzeukHOjtDe3xIg660aYdWKV4bbog9AmkdXL7x0lTkUb+ERVhMCvtIFE7YKGZqeEovL6tgXl9U/ApdXK/xT0283CWoKBQVcvZUEqimtWTaEFekFD4PTDkwfUg6WZY6Gy6yTU4HESziSh5e0raH7mP4YJ8tZsdtnfIL+NRvReUqFz8goG6Dm0nvsvwcI8jJhH8lGbPxd6hqbvk+PnttZRr5uxiIJwIx/98fW+mAL0N7AScRklFSjQgr4dRTqZ+/TXyUj9E0x/nyaEpRuj83SzLSwFsc2izoxNCSJDz3m5t7RW2Inm3X3oZmkFOdWL4Y1yGIHcFY0i9LSgHYaQpfLpDz4WnlkkU8cyf73Ic= rawda@rawda-Inspiron-3576"
      TARGET  = grid_deployment.d1.vms[0].ip
    }
    planetary = true
  }
}

output "computed_public_ip" {
  value = grid_deployment.d2.vms[0].computedip
}
