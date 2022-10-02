module "kubernetes" {
  source  = "../../../modules/k8s-module"
  ssh     = local.ssh
  token   = local.token
  network = local.network
  master  = local.master
  workers = local.workers
  disks   = local.disks
}

locals {
  ssh   = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCs+AFNbOtMtWElFISu1NLke5dH3x+HKJ1Ef6qYpMzZlF9UfzKhcTSy+LQTxvk55dABBirsln03rRdsblmyCgJAPq/w75QVRJCoh8Ge47eOmvaIx6MLFKTVHbfdUTaqFUZ9B6OxnufPc/T/4uWuBHXGZHNu+6DFS6nx7d0hQJtke4fetEzu+6LjIup0V9Qvt2xSK7kTTuDqHbXzvqc8J9PWmhTr0Q5N3qNJ2g8RrTO3Whmb7Pr0qMA4gWuBPEQoDHnb0YuXqxd3L94bqf2dqo8zo1dVwAESe9OCjwFzSw/1XyPoHPzMxN5B1Uu0hgwGlUagRnDg/C/kA6RJBht91Q/fXDWdB/sLVMfGKZ8EiybRynMQcQMVGebVOw5dQyK5Jt069spBmlqZzJZ4Zpa6ktxwFW2foJxObVhm5fmFr6c7PYIyT03OkY83V9DJVFR+HiVi5in+0DOujMDoyQYV/6Zyvs1uMeJHARJTjvEYz6dbYcA7odp3Zi4Tmv+d+3nkx+k= superluigi@luigi"
  token = "838a6db4"

  network = {
    nodes         = [45]
    ip_range      = "10.1.0.0/16"
    name          = "test_network"
    description   = "new network for testing"
    add_wg_access = false
  }

  master = {
    name        = "mr"
    node        = 45
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
      node        = 45
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
      node        = 45
      size        = 5
      description = ""
    },
    {
      name        = "w0disk"
      node        = 45
      size        = 2
      description = ""
    },
  ]
}
