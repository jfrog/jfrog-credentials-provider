# =============================================================
# EXAMPLE CONFIGURATION: CREATE NEW EKS CLUSTER
# =============================================================
# This example creates a completely new EKS cluster with the 
# JFrog Credential Provider pre-configured.

# AWS Region to be used
region = "us-west-2"

# Create a new EKS cluster
create_eks_cluster = true
cluster_name = "jfrog-test-cluster"

# Allow access from your IP addresses (CHANGE THESE TO YOUR IPs)
# WARNING: Do not use "0.0.0.0/0" in production environments
cluster_public_access_cidrs = [
  "203.0.113.0/32",  # Replace with your actual IP
  "198.51.100.0/32"  # Replace with your actual IP
]

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
ami_type = "AL2_x86_64"

# OIDC Configuration (if using OIDC authentication)
jfrog_oidc_provider_name = "jfrog-aws-oidc-provider"
jfrog_namespace = "jfrog"
