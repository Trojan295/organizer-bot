terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }

  backend "s3" {
  }
}

provider "aws" {
  region = var.region
  default_tags {
    tags = local.common_tags
  }
}
