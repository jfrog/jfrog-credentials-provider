# Configure the needed providers
terraform {
  required_providers {
    # AWS provider
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    # Azure providers
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 3.0"
    }
    azapi = {
      source  = "azure/azapi"
      version = "~> 2.0"
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

provider "azurerm" {
  features {
    key_vault {
      purge_soft_delete_on_destroy = true
    }
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
  subscription_id = var.azure_subscription_id
  resource_provider_registrations = "none"
}
