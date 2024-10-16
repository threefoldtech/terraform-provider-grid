terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "random_bytes" "mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "mycelium_key" {
  length = 32
}

resource "grid_scheduler" "sched" {
  requests {
    name             = "node1"
    mru              = 4 * 1024
    cru              = 2
    public_config    = true
    public_ips_count = 1
    yggdrasil        = true
    wireguard        = true
  }
}

locals {
  solution_type = "CosmosValidator"
  name          = "cosmosvalidator"
  node1         = grid_scheduler.sched.nodes["node1"]
}

resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = [local.node1]
  ip_range      = "10.1.0.0/16"
  description   = "newer network"
  add_wg_access = true
  mycelium_keys = {
    format("%s", grid_scheduler.sched.nodes["node1"]) = random_bytes.mycelium_key.hex
  }
}
resource "grid_deployment" "d1" {
  solution_type = local.solution_type
  name          = local.name
  node          = local.node1
  network_name  = grid_network.net1.name
  vms {
    name             = "vm1"
    flist            = "https://hub.grid.tf/tf-official-apps/threefold_hub-latest.flist"
    cpu              = 2
    publicip         = true
    memory           = 4096
    entrypoint       = "/sbin/zinit init"
    mycelium_ip_seed = random_bytes.mycelium_ip_seed.hex
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")

      MNEMONICS         = "<MNEMONICS>"
      KEYNAME           = "ashraaf"
      STAKE_AMOUNT      = "100000000stake"
      MONIKER           = "ashroofdfd"
      CHAIN_ID          = "threefold-hub"
      ETHEREUM_ADDRESS  = "ETHEREUM_ADDRESS"
      ETHEREUM_PRIV_KEY = "<ETHEREUM_PRIVATE_KEY>"
      GRAVITY_ADDRESS   = "GRAVIRY CONTRACT ADDRESS"
      ETHEREUM_RPC      = "http://<IP>:8575"
      PERSISTENT_PEERS  = "780e271b5a835722ba0fac1c979e54d078e57e38@161.35.85.34:26656"
      GENESIS_URL       = "https://gist.githubusercontent.com/ashraffouda/1e494d95ad60ed8f72805c47a0493da7/raw/9955c1488dabd1bdcdeb60f12b1120b1ae3a74ca/genesis.json"
    }
    planetary = true
  }
}
output "wg_config" {
  value = grid_network.net1.access_wg_config
}
output "node1_zmachine1_ip" {
  value = grid_deployment.d1.vms[0].ip
}

output "ygg_ip" {
  value = grid_deployment.d1.vms[0].planetary_ip
}

output "mycelium_ip" {
  value = grid_deployment.d1.vms[0].mycelium_ip
}
