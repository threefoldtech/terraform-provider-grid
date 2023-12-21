variable "names" {
  type = list(any)
  validation {
    condition     = length(var.names) == length(distinct(var.names))
    error_message = "Master and workers names ${var.names} must be distinct"
  }
}
