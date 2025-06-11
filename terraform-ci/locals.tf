# TODO - add random string to node role names to allow simaultaneous runs
# Currently, using random string causes issues with terraform plan
locals {
    cluster_name = var.create_eks_cluster ? var.cluster_name : var.self_managed_eks_cluster["name"]
    node_role_name = "eks-node-role-${local.cluster_name}-${var.region}"
    cognito_user_pool_domain_name = "jfrog-oidc-domain-${var.region}"
    jfrog_oidc_provider_secret = <<-EOT
    {"client-secret":"${aws_cognito_user_pool_client.jfrog_user_pool_client.client_secret}",
    "client-id":"${aws_cognito_user_pool_client.jfrog_user_pool_client.id}"}
    EOT
    iam_role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${aws_iam_role.eks_node_role.name}"
}