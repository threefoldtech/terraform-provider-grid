
resource "grid_capacity_reserver" "master" {
  farm     = 1
  cpu      = 2
  memory   = 2048
  ssd      = 25
  group_id = group.g1.group_id

}
resource "grid_capacity_reserver" "worker1" {
  farm     = 1
  cpu      = 2
  memory   = 2048
  ssd      = 15
  group_id = group.g1.group_id

}

module "kubernetes" {
  source  = "github.com/threefoldtech/terraform-provider-grid/modules/k8s-module"
  ssh     = local.ssh
  token   = local.token
  network = local.network
  master  = local.master
  workers = local.workers
  disks   = local.disks
}

locals {
  ssh   = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC+/mcyN8lmXYY0/8+irXsYpL6uSQHAG/Tulg4O610A3RnUOKt3F42SuTtGDu1uvQX/vdnb+MgXnwLy+zsOe3YISUgvXWJQJOgMvphkisHyfCFeYDE8NyGRpCmlsuKr0jsj3fmyuCAV5TXJWRCKEOxU7wdPUeGC3+VhOFTI7JOHLdT06IX1wznekj+bKUZKbQHV5d4MTHo9dmoQirQU4AyrIMC0K2jHUCMJByLs81evYaplfZmLNbtDW/3KbKa+lh2NovCAbtvu1mC+GgELnOSm7RQ7AEta+a5BEnCEg9sYjZ2PlVt3pihogWtnzkEkd7/cmTk3exrDX86emZSga+NWaI+/mQODpdDsWStetwVIo1WpVdmJLmviPGcwXXx5unDYqFqkJ9F+OnbedCFh/U/9+tSg1/2BsKo81N9zNpoprQCPCKtHgLDbEnHaL7D1Xx2b9/8GD84ADaRr55f34L9mLHvaBRRvZ8L4Jl845KuJ9GCEkmirBHCtdSoIZrWqAbE= islam@islam"
  token = "838a6db4"

  network = {
    capacity_ids  = [grid_capacity_reserver.master.capacity_id, grid_capacity_reserver.worker1.capacity_id]
    ip_range      = "10.1.0.0/16"
    name          = "test_network"
    description   = "new network for testing"
    add_wg_access = false
  }

  master = {
    name        = "mr"
    capacity_id = grid_capacity_reserver.master.capacity_id
    cpu         = 2
    memory      = 1024
    disk_name   = "mrdisk"
    mount_point = "/mydisk"
    publicip    = true
    planetary   = false
  }

  workers = [
    {
      name        = "w0"
      capacity_id = grid_capacity_reserver.worker1.capacity_id
      cpu         = 1
      memory      = 1024
      disk_name   = "w0disk"
      mount_point = "/mydisk"
      publicip    = false
      planetary   = false
    },
  ]

  disks = [
    {
      name        = "mrdisk"
      size        = 5
      description = ""
    },
    {
      name        = "w0disk"
      size        = 2
      description = ""
    },
  ]
}
