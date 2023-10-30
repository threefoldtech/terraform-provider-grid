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
    name      = string
    node      = number
    cpu       = number
    memory    = number
    planetary = bool
  }))

  validation {
    condition     = length(var.servers) == 3
    error_message = "nomad servers should be exactly 3"
  }
}

variable "clients" {
  type = list(object({
    name      = string
    node      = number
    cpu       = number
    memory    = number
    planetary = bool
  }))
}
