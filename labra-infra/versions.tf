terraform {
  #  pinned this so we all run the same Terraform core version
  required_version = ">= 1.6.0"

  #  pinned provider source and range so local plans stay consistent across our machines
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
