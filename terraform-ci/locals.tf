# TODO - add random string to node role names to allow simaultaneous runs
# Currently, using random string causes issues with terraform plan
locals {
    # AWS locals
    cluster_name = var.enable_aws ? (var.create_eks_cluster ? var.cluster_name : var.self_managed_eks_cluster["name"]) : null
    node_role_name = var.enable_aws ? "eks-node-role-${local.cluster_name}-${var.region}" : null
    cognito_user_pool_domain_name = var.enable_aws ? "jfrog-oidc-domain-${local.cluster_name}-${var.region}" : null
    jfrog_oidc_provider_secret = var.enable_aws ? jsonencode({
      "client-secret" = aws_cognito_user_pool_client.jfrog_user_pool_client[0].client_secret
      "client-id"     = aws_cognito_user_pool_client.jfrog_user_pool_client[0].id
    }) : null
    iam_role_arn = var.enable_aws ? "arn:aws:iam::${data.aws_caller_identity.current[0].account_id}:role/${aws_iam_role.eks_node_role[0].name}" : null
    eks_oidc_id = var.enable_aws ? try(split("/id/", var.create_eks_cluster ? module.eks[0].cluster_oidc_issuer_url : data.aws_eks_cluster.eks_cluster_data[0].identity[0].oidc[0].issuer)[1], null) : null
    # Azure locals
    aks_cluster_name = var.enable_azure ? (var.create_aks_cluster ? var.aks_cluster_name : var.aks_cluster_name) : null
}