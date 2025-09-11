# Configure the needed providers
terraform {
  required_providers {
    # AWS provider
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    # Kubernetes provider
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.36.0"
    }
  }
}

provider "aws" {
  region = var.region
  skip_credentials_validation = true
}
