terraform {
  required_providers {
    grid = {
      version = "0.2"
      source  = "ashraffouda.com/edu/grid"
    }
  }
}

provider "grid" {}


resource "grid_disk" "mydisk1" {
  name = "mydisk1"
  size = 1
  description = "this is my disk description1"
}

