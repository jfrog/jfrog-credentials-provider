region = "us-west-2"

# JFrog Artifactory Configuration
artifactory_url = "example.jfrog.io"
artifactory_user = "aws-eks-user"
iam_role_arn = "arn:aws:iam::123456789012:role/jfrog-assume-role"

# EKS Node Group Configuration
node_role_arn = "arn:aws:iam::123456789012:role/eks-node-role"
cluster_name = "my-eks-cluster"
cluster_service_ipv4_cidr = "172.20.0.0/16"
subnet_ids = ["subnet-abc123", "subnet-def456"]

node_groups = [
  {
    name = "jfrog-enabled-ng"
    desired_size = 2
    max_size = 4
    min_size = 1
    ami_type = "AL2023_ARM_64_STANDARD"
    instance_types = ["t3.medium"]
    labels = {
      "jfrog-credential-provider" = "enabled"
    }
  }
]

# JFrog Credential Provider Binary URL
jfrog_credential_provider_binary_url = "https://releases.jfrog.io/artifactory/run/jfrog-credentials-provider/0.0.5/jfrog-credential-provider-aws-linux"
