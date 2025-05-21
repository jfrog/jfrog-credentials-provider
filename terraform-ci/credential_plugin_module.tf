module manage_eks_nodes_using_jfrog_credential_plugin {
    source = "terraform-prod"
    region = "ap-northeast-3"

    jfrog_credential_provider_binary_url = "https://eldada.jfrog.io/artifactory/public-local/jfrog_credentials_provider/jfrog-credential-provider-aws-linux"
    artifactory_url  = "partnership.jfrog.io"

    artifactory_user = "aws-eks-user"

    artifactory_oidc_identity_mapping_username = "robind"

    create_eks_node_groups = true

    authentication_method = "cognito_oidc"

    eks_node_group_configuration = {
        node_role_arn                   = "arn:aws:iam::095132750011:role/OperatorSelfManagedWorkerNodeRole"
        cluster_name                    = "aws-operator-jfrog"
        cluster_service_ipv4_cidr       = "10.100.0.0/16"
        subnet_ids                      = ["subnet-048aa0af029b4ee79", "subnet-0ed46afcf510b0f80", "subnet-0dde3d3edf430d24b"]
        node_groups = [
            {
                name            = "jfrog-credential-plugin-arm64"
                desired_size    = 1
                max_size        = 2
                min_size        = 1
                ami_type        = "AL2_ARM_64"
                instance_types  = ["t4g.medium"]
                labels          = {
                    createdBy = "kubelet-plugin-test-ci",
                    nodeType = "cognito-oidc"
                }
                taints          = [
                    {
                        key    = "jfrog-kubelet-oidc-ng"
                        value  = "true"
                        effect = "NO_SCHEDULE"
                    }
                ]
            }
        ]
    }

}