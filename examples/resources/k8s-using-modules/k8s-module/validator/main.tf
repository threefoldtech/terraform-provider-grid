variable "names" {
    type = list
    validation {
        condition = length(var.names) == length(distinct(var.names))
        error_message = "Master and workers names must be distinct"
    }
}
