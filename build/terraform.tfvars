region = "ap-northeast-3"

# The JFrog Artifactory URL (the one that will be the EKS container registry)
artifactory_url  = "partnership.jfrog.io"

# The JFrog Artifactory username that will be granted the assume role permission
artifactory_user = "aws-eks-user"

create_eks_cluster = false

self_managed_eks_cluster = {
    name = "aws-operator-jfrog"
}

artifactory_oidc_identity_mapping_username = "robind"