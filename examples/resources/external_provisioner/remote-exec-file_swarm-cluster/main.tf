terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}

provider "grid" {
}

resource "random_bytes" "vm1_mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "vm2_mycelium_ip_seed" {
  length = 6
}

resource "random_bytes" "vm1_mycelium_key" {
  length = 32
}

resource "random_bytes" "vm2_mycelium_key" {
  length = 32
}

resource "grid_scheduler" "sched" {
  requests {
    name      = "node1"
    cru       = 4
    sru       = 1024
    mru       = 2048
    yggdrasil = false
    wireguard = true
  }
}

locals {
  name    = "myvm"
  node_id = grid_scheduler.sched.nodes["node1"]
}

resource "grid_network" "net1" {
  nodes         = [local.node_id]
  ip_range      = "10.1.0.0/16"
  name          = local.name
  description   = "newer network"
  add_wg_access = true
  mycelium_keys = {
    format("%s", local.node_id) = random_bytes.vm1_mycelium_key.hex
    format("%s", local.node_id) = random_bytes.vm2_mycelium_key.hex
  }
}

resource "grid_deployment" "swarm1" {
  name         = local.name
  node         = local.node_id
  network_name = grid_network.net1.name
  vms {
    name        = "swarmManager1"
    flist       = "https://hub.grid.tf/tf-official-apps/grid3_ubuntu20.04_debug-latest.flist"
    entrypoint  = "/init.sh"
    cpu         = 2
    memory      = 1024
    rootfs_size = 25000
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
    mycelium_ip_seed = random_bytes.vm1_mycelium_ip_seed.hex
  }
  vms {
    name        = "swarmWorker1"
    flist       = "https://hub.grid.tf/tf-official-apps/grid3_ubuntu20.04_debug-latest.flist"
    entrypoint  = "/init.sh"
    cpu         = 2
    memory      = 1024
    rootfs_size = 25000
    env_vars = {
      SSH_KEY = file("~/.ssh/id_rsa.pub")
    }
    mycelium_ip_seed = random_bytes.vm2_mycelium_ip_seed.hex
  }

  provisioner "remote-exec" {
    inline = [
      "curl -fsSL https://get.docker.com/ | sh",
      "setsid /usr/bin/containerd &",
      "setsid /usr/bin/dockerd -H unix:// --containerd=/run/containerd/containerd.sock &",
      "sleep 10",
      "docker swarm init --advertise-addr ${self.vms[0].mycelium_ip}",
      "docker swarm join-token --quiet worker > /root/token",
    ]
    connection {
      type    = "ssh"
      user    = "root"
      agent   = true
      host    = self.vms[0].mycelium_ip
      timeout = "20s"
    }
  }

  provisioner "file" {
    source      = "~/.ssh/id_rsa"
    destination = "/root/.ssh/id_rsa"
    connection {
      type    = "ssh"
      user    = "root"
      agent   = true
      host    = self.vms[1].mycelium_ip
      timeout = "20s"
    }
  }

  provisioner "remote-exec" {
    inline = [
      "curl -fsSL https://get.docker.com/ | sh",
      "setsid /usr/bin/containerd &",
      "setsid /usr/bin/dockerd -H unix:// --containerd=/run/containerd/containerd.sock &",
      "chmod 400 /root/.ssh/id_rsa",
      "scp -o StrictHostKeyChecking=no -o NoHostAuthenticationForLocalhost=yes -o UserKnownHostsFile=/dev/null -i /root/.ssh/id_rsa root@[${self.vms[0].mycelium_ip}]:/root/token .",
      "docker swarm join --token $(cat /root/token) [${self.vms[0].mycelium_ip}]:2377"
    ]
    connection {
      type        = "ssh"
      user        = "root"
      agent       = true
      host        = self.vms[0].mycelium_ip
      timeout     = "20s"
      private_key = file("~/.ssh/id_rsa")
    }
  }
}



output "node1_zmachine1_ip" {
  value = grid_deployment.swarm1.vms[0].ip
}

output "node1_zmachine2_ip" {
  value = grid_deployment.swarm1.vms[1].ip
}

output "node1_zmachine1_mycelium_ip" {
  value = grid_deployment.swarm1.vms[0].mycelium_ip
}

output "node1_zmachine2_mycelium_ip" {
  value = grid_deployment.swarm1.vms[1].mycelium_ip
}
