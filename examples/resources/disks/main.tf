terraform {
  required_providers {
    grid = {
      version = "0.1"
      source  = "threefoldtech.com/providers/grid"
    }
  }
}

provider "grid" {    
}

resource "grid_deployment" "d1" {
  node = 1
  
  zdbs{
    name = "zdb1"
    size = 1
    description = "zdb1 description"
    password = "zdbpasswd1"
    mode = "user"
  }
  zdbs{
    name = "zdb2"
    size = 2
    description = "zdb2 description"
    password = "zdbpasswd2"
    mode = "seq"
  }
}

output "deployment_id" {
    value = grid_deployment.d1.id
}
