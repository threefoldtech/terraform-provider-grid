# Debugging steps when terraform is not running

## First time running terraform

You can skip redis and yggdrasil steps if you opt in to using the rmb proxy (USE_RMB_PROXY is true which is the default starting from v0.1.10)

1. make sure you have redis running.
2. check if yggdrasil is up and running with some peers added, and you can reach services listed [here](https://yggdrasil-network.github.io/services.html).
3. check for firewall settings that may prevent other clients to reach you on yggdrasil ip on port 8051.
4. check that your polka account is funded.
5. check that your polka account have key type ed25519.

## You have run it before

1. Make sure you added `-parallelism=1` flag.
2. Check the current plugin [limitation](https://github.com/threefoldtech/terraform-provider-grid#current-limitation), open an issue if it's not one of them. Make sure to include the output of running the plugin with TF_LOG=DEBUG (e.g. `TF_LOG=DEBUG terraform apply -parallelism=1`)