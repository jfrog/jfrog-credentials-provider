region = "ap-northeast-3"

# The JFrog Credential Provider binary URL (no authentication required)
# added by CI
# jfrog_credential_provider_binary_url = "https://releases.jfrog.io/artifactory/run/jfrog-credentials-provider/0.1.0-beta.1/jfrog-credential-provider-aws-linux"

# The JFrog Artifactory URL (the one that will be the EKS container registry)
artifactory_url  = "partnership.jfrog.io"
# Change this to jfrogurl


# The JFrog Artifactory username that will be granted the assume role permission
artifactory_user = "aws-eks-user"

create_eks_cluster = false
# cluster_public_access_cidrs = ["0.0.0.0/0"]
# cluster_name = "demo-eks-cluster"

self_managed_eks_cluster = {
    name = "aws-operator-jfrog"
}

jfrog_namespace = "jfrog"