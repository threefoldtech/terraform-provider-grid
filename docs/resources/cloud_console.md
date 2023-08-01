cloud console requirements :
-terraform or ts-client
-wireguard
steps to produce:

1. make a deployment thought terraform single_vm and let the wire guard access in network options and cloud console in deployment options with True.
![image](https://github.com/threefoldtech/terraform-provider-grid/assets/110984055/04ca0f73-d532-4a76-8888-f564de757a32)
2. make sure to get the output of both.
![image](https://github.com/threefoldtech/terraform-provider-grid/assets/110984055/345a035d-5b8b-4d87-aa63-67ebcd965a7a)
![image](https://github.com/threefoldtech/terraform-provider-grid/assets/110984055/6fb2067f-b0e9-4fbc-afe2-3c79e8b8d52e)
3. after using `terraform init && terraform apply` and the deployment is made successfully you will get the wireguard interface file successfully and cloud console url.
![image](https://github.com/threefoldtech/terraform-provider-grid/assets/110984055/9d7a9d21-9bc7-4d90-9784-6633a266db81)
4. copy the wire-guard interface and paste in new text file and make sure to be named <example>.conf
![image](https://github.com/threefoldtech/terraform-provider-grid/assets/110984055/c87e2581-fe62-41dd-8fa9-8877142ba094)
5. use `sudo wg-quick up ./examble.conf`
6. then run `ip a` and check that ip is added with success.
7. if ip is not added check if in the 'ip a' list if any ip makes a conflict with the wireguard ip and use `ifconfig <conflict_ip> down` and then repeat step 5
8. now open the cloud console url to get ypur online console for the vm.
![image](https://github.com/threefoldtech/terraform-provider-grid/assets/110984055/435e2b72-0b48-46f2-b9c4-d61bafd63d52)
