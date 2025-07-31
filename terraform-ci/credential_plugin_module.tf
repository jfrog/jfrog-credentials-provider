module manage_eks_nodes_using_jfrog_credential_plugin {
    source = "./../terraform-module"
    region = var.region

    jfrog_credential_provider_binary_url = var.jfrog_credential_provider_binary_url
    artifactory_url  = var.artifactory_url

    artifactory_user = var.artifactory_user

    create_eks_node_groups = true

    authentication_method = "assume_role"

    iam_role_arn = local.iam_role_arn

    wait_for_creation = aws_iam_role.eks_node_role.arn

    eks_node_group_configuration = {
        node_role_arn                   = local.iam_role_arn
        cluster_name                    = !var.create_eks_cluster ? var.self_managed_eks_cluster.name : module.eks[0].cluster_name
        cluster_service_ipv4_cidr       = data.aws_eks_cluster.eks_cluster_data.kubernetes_network_config[0].service_ipv4_cidr
        subnet_ids                      = data.aws_eks_cluster.eks_cluster_data.vpc_config[0].subnet_ids
        node_groups = [
            {
                name            = "jfrog-credential-plugin-arm64"
                vpc_security_group_ids = var.create_eks_cluster ? [module.eks[0].node_security_group_id] : var.node_security_group_ids
                desired_size    = var.node_group_desired_size
                max_size        = var.node_group_max_size
                min_size        = var.node_group_min_size
                ami_type        = var.ami_type
                instance_types  = var.node_group_instance_types
                labels          = {
                    createdBy = "kubelet-plugin-test-ci",
                    nodeType = "cognito-oidc"
                }
                taints          = {
                    oidcTaint = {
                        key    = "jfrog-kubelet-oidc-ng"
                        value  = "true"
                        effect = "NO_SCHEDULE"
                    }
                }
            }
        ]
    }
}

module create_daemonset_with_plugin_enabled {
    source = "./../terraform-module"
    region = var.region

    jfrog_credential_provider_binary_url = var.jfrog_credential_provider_binary_url
    artifactory_url  = var.artifactory_url

    artifactory_user = var.artifactory_user

    create_eks_node_groups = false
    jfrog_credential_plugin_daemonset_installation = true

    authentication_method = "cognito_oidc"

    iam_role_arn = local.iam_role_arn

    wait_for_creation = aws_iam_role.eks_node_role.arn

    kubernetes_auth_object  = {
        host                   = data.aws_eks_cluster.eks_cluster_data.endpoint
        cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks_cluster_data.certificate_authority.0.data)
        token                  = data.aws_eks_cluster_auth.eks_cluster_auth.token
    }

    # OIDC Configuration (Required when authentication_method = "cognito_oidc")
    jfrog_oidc_provider_name = var.jfrog_oidc_provider_name
    aws_cognito_user_pool_name = aws_cognito_user_pool.jfrog_cognito_user_pool.name
    aws_cognito_user_pool_id = aws_cognito_user_pool.jfrog_cognito_user_pool.id
    aws_cognito_user_pool_client_id = aws_cognito_user_pool_client.jfrog_user_pool_client.id
    aws_cognito_resource_server_name = aws_cognito_resource_server.jfrog_oidc_resource.name
    aws_cognito_user_pool_domain_name = local.cognito_user_pool_domain_name
    aws_cognito_user_pool_secret_name = aws_secretsmanager_secret.jfrog_oidc_integration_secret.name

    daemonset_configuration = {
        jfrog_namespace = var.jfrog_namespace
        node_selector   = [{
            key: "onlyForDaemonset"
            value: "true"
        }]
        tolerations          = [
            {
                key    = "forDaemonset"
                operator = "Equal"
                value  = "true"
                effect = "NoSchedule"
            }
        ]
    }
}