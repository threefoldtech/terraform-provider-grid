# using gatway name on private networks (wireguard)

It is possible to create a vm with private ip (wireguard) and use it as a backend for a gateway contract. this is done as the following

- Create a gateway domain data source. this data source will construct the full domain so we can use it afterwards

```
data "grid_gateway_domain" "domain" {
  node = grid_scheduler.sched.nodes["node1"]
  name = "examp123456"
}
```

- create a network resource

```
resource "grid_network" "net1" {
  nodes       = [grid_scheduler.sched.nodes["node1"]]
  ip>_range    = "10.1.0.0/16"
  name        = mynet
  description = "newer network"
}
```

- Create a vm to host your service

```
resource "grid_deployment" "d1" {
  name         = vm1
  node         = grid_scheduler.sched.nodes["node1"]
  network_name = grid_network.net1.name
  vms {
    ...
  }
}
```

- Create a grid_name_proxy resource using the network created above and the wireguard ip of the vm that host the service. Also consider changing the port to the correct port

```
resource "grid_name_proxy" "p1" {
  node            = grid_scheduler.sched.nodes["node1"]
  name            = "examp123456"
  backends        = [format("http://%s:9000", grid_deployment.d1.vms[0].ip)]
  network         = grid_network.net1.name
  tls_passthrough = false
}
```

- To know the full domain created using the data source above you can show it via

```
output "fqdn" {
  value = data.grid_gateway_domain.domain.fqdn
}
```

- Now vist the domain you should be able to reach your service hosted on the vm
