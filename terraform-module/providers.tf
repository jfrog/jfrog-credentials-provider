# Configure the needed providers
terraform {
  required_providers {
    # AWS provider
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.97.0"
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
}
