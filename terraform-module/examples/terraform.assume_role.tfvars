# =============================================================
# EXAMPLE CONFIGURATION WITH ASSUME ROLE AUTHENTICATION
# =============================================================
region = "us-west-2"

# Deployment Method: Create EKS Node Groups (Uncomment one deployment method only)
create_eks_node_groups = true
# jfrog_credential_plugin_daemonset_installation = false
# generate_aws_cli_command = false

# Authentication Method: Assume Role
authentication_method = "assume_role"
artifactory_url = "example.jfrog.io"
artifactory_user = "aws-eks-user"
iam_role_arn = "arn:aws:iam::123456789012:role/jfrog-assume-role"

# EKS Node Group Configuration (Required when create_eks_node_groups = true)
eks_node_group_configuration = {
  node_role_arn             = "arn:aws:iam::123456789012:role/eks-node-role"
  cluster_name              = "my-eks-cluster"
  cluster_service_ipv4_cidr = "172.20.0.0/16"
  subnet_ids                = ["subnet-abc123", "subnet-def456"]
  
  node_groups = [{
    name           = "jfrog-enabled-ng"
    desired_size   = 2
    max_size       = 4
    min_size       = 1
    ami_type       = "AL2023_ARM_64_STANDARD"
    instance_types = ["t3.medium"]
    labels         = {
      "jfrog-credential-provider" = "enabled"
    }
  }]
}

# DaemonSet Configuration (Required when jfrog_credential_plugin_daemonset_installation = true)
# daemonset_configuration = {
#   jfrog_namespace = "jfrog"
#   node_selector = [
#     {
#       key = "kubernetes.io/os"
#       value = "linux"
#     }
#   ]
#   tolerations = [
#     {
#       key      = "dedicated"
#       operator = "Equal"
#       value    = "jfrog"
#       effect   = "NoSchedule"
#     }
#   ]
# }
# kubeconfig_path = "~/.kube/config"

# JFrog Credential Provider Binary URL
jfrog_credential_provider_binary_url = "https://releases.jfrog.io/artifactory/run/jfrog-credentials-provider/0.1.0-beta.1/jfrog-credential-provider-aws-linux"

# OIDC Configuration (Required when authentication_method = "cognito_oidc")
# jfrog_oidc_provider_name = "jfrog-aws-oidc-provider"
# aws_cognito_user_pool_name = "jfrog-user-pool"
# aws_cognito_user_pool_id = "ap-northeast-3_rasdsada"
# aws_cognito_user_pool_client_id = "random11"
# aws_cognito_resource_server_name = "jfrog-resource-server"
# aws_cognito_user_pool_domain_name = "jfrog-domain"
# aws_cognito_user_pool_secret_name = "jfrog-cognito-credentials"
# artifactory_oidc_identity_mapping_username = "oidc-mapped-user"
