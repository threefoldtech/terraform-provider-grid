terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}
locals {
  solution_type = "CosmosValidator"
  name          = "cosmosvalidator"
}
resource "grid_network" "net1" {
  solution_type = local.solution_type
  name          = local.name
  nodes         = [8]
  ip_range      = "10.1.0.0/16"
  description   = "newer network"
  add_wg_access = true
}
resource "grid_deployment" "d1" {
  solution_type = local.solution_type
  name          = local.name
  node          = 8
  network_name  = grid_network.net1.name
  ip_range      = lookup(grid_network.net1.nodes_ip_range, 8, "")
  vms {
    name       = "vm1"
    flist      = "https://hub.grid.tf/tf-official-apps/threefold_hub-latest.flist"
    cpu        = 2
    publicip   = true
    memory     = 4096
    entrypoint = "/sbin/zinit init"
    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP ashraf@thinkpad"

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
  value = grid_deployment.d1.vms[0].ygg_ip
}
