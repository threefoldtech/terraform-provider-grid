variable "ssh" {
  type = string
}

variable "token" {
  type = string
}

variable "network" {
  type = object({
    nodes         = list(number)
    name          = string
    ip_range      = string
    description   = string
    add_wg_access = bool
  })
}

variable "master" {
  type = object({
    name        = string
    node        = number
    cpu         = number
    memory      = number
    disk_name   = string
    mount_point = string
    publicip    = bool
    planetary   = bool
  })

  validation {
    condition     = var.master.memory >= 1024
    error_message = "Memory must be at least 1GB not ${var.master.memory}MB"
  }
}

variable "workers" {
  type = list(object({
    name        = string
    node        = number
    cpu         = number
    memory      = number
    disk_name   = string
    mount_point = string
    publicip    = bool
    planetary   = bool
  }))

  validation {
    condition = ([
      for w in var.workers : true if w.memory >= 1024
    ]) != length(var.workers)
    error_message = "Memory must be at least 1GB not ${var.master.memory}MB"
  }
}

variable "disks" {
  type = list(object({
    node        = number
    name        = string
    size        = number
    description = string
  }))
}
