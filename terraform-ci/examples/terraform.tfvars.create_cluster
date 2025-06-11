# =============================================================
# EXAMPLE CONFIGURATION: CREATE NEW EKS CLUSTER
# =============================================================
# This example creates a completely new EKS cluster
# Creates a node group and daemonset in it that configures the JFrog Credential Provider

# Node Group Configuration
# Change if needed
# node_group_desired_size = 2
# node_group_max_size = 4
# node_group_min_size = 1
# node_group_instance_types = ["t3.medium"]
# ami_type = "AL2023_ARM_64_STANDARD"


jfrog_namespace = "jfrog"

region = "ap-northeast-3"

# The JFrog Credential Provider binary URL (no authentication required)
jfrog_credential_provider_binary_url = "<download-url>"

# The JFrog Artifactory URL (the one that will be the EKS container registry)
artifactory_url  = "<artifactory-url>"
# Change this to jfrogurl

# The JFrog Artifactory username that will be granted the assume role permission
artifactory_user = "aws-eks-user"

create_eks_cluster = true
# cluster_public_access_cidrs = ["0.0.0.0/0"]
# cluster_name = "demo-eks-cluster"

self_managed_eks_cluster = {
    name = "aws-operator-jfrog"
}
