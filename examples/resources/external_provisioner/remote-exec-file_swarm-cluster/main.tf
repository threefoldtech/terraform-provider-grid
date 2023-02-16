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
  name = "myvm"
}

resource "grid_network" "net1" {
  nodes         = [1]
  ip_range      = "10.1.0.0/16"
  name          = local.name
  description   = "newer network"
  add_wg_access = true
}

resource "grid_deployment" "swarm1" {
  name         = local.name
  node         = 1
  network_name = grid_network.net1.name
  vms {
    name        = "swarmManager1"
    flist       = "https://hub.grid.tf/tf-official-apps/grid3_ubuntu20.04_debug-latest.flist"
    entrypoint  = "/init.sh"
    cpu         = 2
    memory      = 1024
    rootfs_size = 25000
    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC9MI7fh4xEOOEKL7PvLvXmSeRWesToj6E26bbDASvlZnyzlSKFLuYRpnVjkr8JcuWKZP6RQn8+2aRs6Owyx7Tx+9kmEh7WI5fol0JNDn1D0gjp4XtGnqnON7d0d5oFI+EjQQwgCZwvg0PnV/2DYoH4GJ6KPCclPz4a6eXrblCLA2CHTzghDgyj2x5B4vB3rtoI/GAYYNqxB7REngOG6hct8vdtSndeY1sxuRoBnophf7MPHklRQ6EG2GxQVzAOsBgGHWSJPsXQkxbs8am0C9uEDL+BJuSyFbc/fSRKptU1UmS18kdEjRgGNoQD7D+Maxh1EbmudYqKW92TVgdxXWTQv1b1+3dG5+9g+hIWkbKZCBcfMe4nA5H7qerLvoFWLl6dKhayt1xx5mv8XhXCpEC22/XHxhRBHBaWwSSI+QPOCvs4cdrn4sQU+EXsy7+T7FIXPeWiC2jhFd6j8WIHAv6/rRPsiwV1dobzZOrCxTOnrqPB+756t7ANxuktsVlAZaM= sameh@sameh-inspiron-3576"
    }
    planetary = true
  }
  vms {
    name        = "swarmWorker1"
    flist       = "https://hub.grid.tf/tf-official-apps/grid3_ubuntu20.04_debug-latest.flist"
    entrypoint  = "/init.sh"
    cpu         = 2
    memory      = 1024
    rootfs_size = 25000
    env_vars = {
      SSH_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC9MI7fh4xEOOEKL7PvLvXmSeRWesToj6E26bbDASvlZnyzlSKFLuYRpnVjkr8JcuWKZP6RQn8+2aRs6Owyx7Tx+9kmEh7WI5fol0JNDn1D0gjp4XtGnqnON7d0d5oFI+EjQQwgCZwvg0PnV/2DYoH4GJ6KPCclPz4a6eXrblCLA2CHTzghDgyj2x5B4vB3rtoI/GAYYNqxB7REngOG6hct8vdtSndeY1sxuRoBnophf7MPHklRQ6EG2GxQVzAOsBgGHWSJPsXQkxbs8am0C9uEDL+BJuSyFbc/fSRKptU1UmS18kdEjRgGNoQD7D+Maxh1EbmudYqKW92TVgdxXWTQv1b1+3dG5+9g+hIWkbKZCBcfMe4nA5H7qerLvoFWLl6dKhayt1xx5mv8XhXCpEC22/XHxhRBHBaWwSSI+QPOCvs4cdrn4sQU+EXsy7+T7FIXPeWiC2jhFd6j8WIHAv6/rRPsiwV1dobzZOrCxTOnrqPB+756t7ANxuktsVlAZaM= sameh@sameh-inspiron-3576"
    }
    planetary = true
  }

  provisioner "remote-exec" {
    inline = [
      "curl -fsSL https://get.docker.com/ | sh",
      "setsid /usr/bin/containerd &",
      "setsid /usr/bin/dockerd -H unix:// --containerd=/run/containerd/containerd.sock &",
      "sleep 10",
      "docker swarm init --advertise-addr ${grid_deployment.swarm1.vms[0].ygg_ip}",
      "docker swarm join-token --quiet worker > /root/token",
    ]
    connection {
      type    = "ssh"
      user    = "root"
      agent   = true
      host    = grid_deployment.swarm1.vms[0].ygg_ip
      timeout = "20s"
    }
  }

  provisioner "file" {
    source      = "/home/sameh/.ssh/id_rsa"
    destination = "/root/.ssh/id_rsa"
    connection {
      type    = "ssh"
      user    = "root"
      agent   = true
      host    = grid_deployment.swarm1.vms[1].ygg_ip
      timeout = "20s"
    }
  }


  provisioner "remote-exec" {
    inline = [
      "curl -fsSL https://get.docker.com/ | sh",
      "setsid /usr/bin/containerd &",
      "setsid /usr/bin/dockerd -H unix:// --containerd=/run/containerd/containerd.sock &",
      "chmod 400 /root/.ssh/id_rsa",
      "scp -o StrictHostKeyChecking=no -o NoHostAuthenticationForLocalhost=yes -o UserKnownHostsFile=/dev/null -i /root/.ssh/id_rsa root@[${grid_deployment.swarm1.vms[0].ygg_ip}]:/root/token .",
      "docker swarm join --token $(cat /root/token) [${grid_deployment.swarm1.vms[0].ygg_ip}]:2377"
    ]
    connection {
      type    = "ssh"
      user    = "root"
      agent   = true
      host    = grid_deployment.swarm1.vms[1].ygg_ip
      timeout = "20s"
    }
  }
}



output "node1_zmachine1_ip" {
  value = grid_deployment.swarm1.vms[0].ip
}

output "node1_zmachine2_ip" {
  value = grid_deployment.swarm1.vms[1].ip
}

output "node1_zmachine1_ygg_ip" {
  value = grid_deployment.swarm1.vms[0].ygg_ip
}

output "node1_zmachine2_ygg_ip" {
  value = grid_deployment.swarm1.vms[1].ygg_ip
}