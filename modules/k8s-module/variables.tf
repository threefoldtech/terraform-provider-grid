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
    mycelium_keys = map(string)
  })
}

variable "master" {
  type = object({
    name             = string
    node             = number
    cpu              = number
    memory           = number
    disk_name        = string
    mount_point      = string
    publicip         = bool
    planetary        = bool
    mycelium_ip_seed = string
  })

  validation {
    condition     = var.master.memory >= 1024
    error_message = "Memory must be at least 1GB"
  }
}

variable "workers" {
  type = list(object({
    name             = string
    node             = number
    cpu              = number
    memory           = number
    disk_name        = string
    mount_point      = string
    publicip         = bool
    planetary        = bool
    mycelium_ip_seed = string
  }))

  validation {
    condition = ([
      for w in var.workers : true if w.memory >= 1024
    ]) != length(var.workers)
    error_message = "Memory must be at least 1GB"
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
