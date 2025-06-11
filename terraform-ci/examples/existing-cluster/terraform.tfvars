# =============================================================
# EXAMPLE CONFIGURATION: USE EXISTING EKS CLUSTER
# =============================================================
# This example uses an existing EKS cluster and adds the 
# JFrog Credential Provider to it.

# AWS Region to be used
region = "us-west-2"

# Use existing EKS cluster
create_eks_cluster = false
self_managed_eks_cluster = {
  name = "my-existing-cluster"
}

# JFrog Configuration
# The JFrog Credential Provider binary URL (no authentication required)
jfrog_credential_provider_binary_url = "https://github.com/jfrog/jfrog-credentials-provider/releases/download/v0.1.0/jfrog-credential-provider-aws-linux-arm64"

# The JFrog Artifactory URL (the one that will be the EKS container registry)
artifactory_url = "example.jfrog.io"

# The JFrog Artifactory username that will be granted the assume role permission
artifactory_user = "aws-eks-user"

# Node Group Configuration
oidc_node_group_desired_size = 2
oidc_node_group_max_size = 4
oidc_node_group_min_size = 1
oidc_node_group_instance_types = ["t3.medium"]
ami_type = "AL2023_ARM_64_STANDARD"

# OIDC Configuration (if using OIDC authentication)
jfrog_oidc_provider_name = "jfrog-aws-oidc-provider"
jfrog_namespace = "jfrog"
