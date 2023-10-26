variable "ssh" {
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

variable "nomad" {
  type = object({
    name         = string
    nodes        = list(number)
    network_name = string

    servers = list(object({
      name        = string
      flist       = string
      cpu         = number
      memory      = number
      entry_point = string
      planetary   = bool
    }))

    clients = list(object({
      name        = string
      flist       = string
      cpu         = number
      memory      = number
      entry_point = string
      planetary   = bool
    }))
  })

  validation {
    condition     = length(var.nomad.servers) == 3
    error_message = "Memory must be at least 1GB"
  }
}
