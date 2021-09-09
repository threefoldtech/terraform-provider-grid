# Debugging steps when terraform is not running

## First time running terraform

1. make sure you have redis running.
2. check if yggdrasil is up and running with some peers added, and you can reach services listed [here](https://yggdrasil-network.github.io/services.html).
3. check for firewall settings that may prevent other clients to reach you on yggdrasil ip.
   - get you yggdrasil ip using `sudo yggdrasil -address -useconffile /etc/yggdrasil.conf` (update the config file if using something other than the default)
   - make sure an yggdrasil-enabled host outside you network can ping you, and can reach you on http://[your-ip]:8051/ (should get "404 not found")
   - if he can't make sure to add rules allowing connections to this endpoint.
4. check that your polka account is funded.
5. check that your polka account have key type ed25519.
6. make sure that you entered the correct `MNEMONICS` and `TWIN_ID`. And `https://explorer.devnet.grid.tf/twin/<your-twin-id>` matches your account id on polka, and the ip matches your yggdrasil ip.

## You have run it before

1. If you get `all retries done`, that means 99% of the time you couldn't some node. Make sure your yggdrasil is working and run again with TF_LOG=DEBUG (e.g. `TF_LOG=DEBUG terraform apply -parallelism=1`), you should see the error that caused this message.
2. Make sure you added `-parallelism=1` flag.
3. Check the current plugin [limitation](https://github.com/threefoldtech/terraform-provider-grid#current-limitation), open an issue if it's not one of them.