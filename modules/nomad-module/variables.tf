variable "ssh_key" {
  type = string
}

variable "first_server_ip" {
  type = string
}

variable "network" {
  type = object({
    nodes       = list(number)
    name        = string
    ip_range    = string
    description = string
  })
}

variable "servers" {
  type = list(object({
    name        = string
    node        = number
    cpu         = optional(number, 2)
    memory      = optional(number, 1024)
    mount_point = optional(string, "/mnt")
    publicip    = optional(bool, false)
    publicip6   = optional(bool, false)
    planetary   = optional(bool, true)
    disk = object({
      name = string
      size = optional(number, 5)
    })
  }))
}

variable "clients" {
  type = list(object({
    name        = string
    node        = number
    cpu         = optional(number, 2)
    memory      = optional(number, 1024)
    mount_point = optional(string, "/mnt")
    publicip    = optional(bool, false)
    publicip6   = optional(bool, false)
    planetary   = optional(bool, true)
    disk = optional(object({
      name = string
      size = optional(number, 5)
    }))
  }))
}
