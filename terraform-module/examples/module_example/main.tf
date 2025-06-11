terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

module "jfrog_credential_provider" {
  source = "../../terraform-module"

  # Deployment Method: EKS Node Group Creation
  create_eks_node_groups = true
  jfrog_credential_plugin_daemonset_installation = false
  generate_aws_cli_command = false

  # Authentication Method: Assume Role
  authentication_method = "assume_role"
  artifactory_url = var.artifactory_url
  artifactory_user = var.artifactory_user
  iam_role_arn = var.iam_role_arn

  # EKS Node Group Configuration
  eks_node_group_configuration = {
    node_role_arn             = var.node_role_arn
    cluster_name              = var.cluster_name
    cluster_service_ipv4_cidr = var.cluster_service_ipv4_cidr
    subnet_ids                = var.subnet_ids
    
    node_groups = var.node_groups
  }

  # JFrog Credential Provider Binary URL
  jfrog_credential_provider_binary_url = var.jfrog_credential_provider_binary_url
}
