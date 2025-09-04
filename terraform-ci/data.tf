# AWS Data Sources
data "aws_availability_zones" "available" {
    count = var.enable_aws ? 1 : 0
    filter {
        name   = "opt-in-status"
        values = ["opt-in-not-required"]
    }
}

data "aws_eks_cluster" "eks_cluster_data" {
    count = var.enable_aws ? 1 : 0
    name = !var.create_eks_cluster ? var.self_managed_eks_cluster["name"] : module.eks[0].cluster_name

    depends_on = [module.eks]
}

data "aws_eks_cluster_auth" "eks_cluster_auth" {
    count = var.enable_aws ? 1 : 0
    name = data.aws_eks_cluster.eks_cluster_data[0].name
}

data "aws_caller_identity" "current" {
    count = var.enable_aws ? 1 : 0
}

# Azure Data Sources
data "azuread_client_config" "current" {
  count = var.enable_azure ? 1 : 0
}

# Data source to get managed identity created by AKS for the agent pool
# data "azurerm_user_assigned_identity" "agentpool_identity" {
#   count = var.enable_azure ? 1 : 0
  
#   name                = "${local.aks_cluster_name}-agentpool"
#   resource_group_name = "MC_${var.azure_resource_group_name}_${local.aks_cluster_name}_${var.azure_location}"
  
#   depends_on = [azurerm_kubernetes_cluster.k8s]
# }