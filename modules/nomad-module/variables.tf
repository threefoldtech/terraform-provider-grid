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

variable "nomad" {
  type = object({
    name         = string
    node        = number
    network_name = string

    servers = list(object({
      name        = string
      cpu         = number
      memory      = number
      planetary   = bool
    }))

    clients = list(object({
      name        = string
      cpu         = number
      memory      = number
      planetary   = bool
    }))
  })

  validation {
    condition     = length(var.nomad.servers) == 3
    error_message = "nomad servers should be exactly 3"
  }
}
