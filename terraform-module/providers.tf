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
  skip_metadata_api_check = true
  skip_region_validation = true
  skip_requesting_account_id = true

  # create a dynamic block to set the access_key, secret_key, and token if they are not null
  access_key = var.enable_aws ? null : "foo"
  secret_key = var.enable_aws ? null : "bar"
  
}
