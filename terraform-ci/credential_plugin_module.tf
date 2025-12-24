module manage_eks_nodes_using_jfrog_credential_plugin {
    enable_aws = var.enable_aws
    source = "./../terraform-module"
    region = var.region

    jfrog_credential_provider_binary_url = var.jfrog_credential_provider_binary_url
    artifactory_url  = var.artifactory_url

    artifactory_glob_pattern = var.artifactory_glob_pattern

    artifactory_user = var.artifactory_user

    create_eks_node_groups = true

    authentication_method = "assume_role"

    iam_role_arn = var.enable_aws ? local.iam_role_arn : null

    wait_for_creation = var.enable_aws ? aws_iam_role.eks_node_role[0].arn : ""

    eks_node_group_configuration = var.enable_aws ? {
        node_role_arn                   = local.iam_role_arn
        cluster_name                    = !var.create_eks_cluster ? var.self_managed_eks_cluster.name : module.eks[0].cluster_name
        cluster_service_ipv4_cidr       = data.aws_eks_cluster.eks_cluster_data[0].kubernetes_network_config[0].service_ipv4_cidr
        subnet_ids                      = data.aws_eks_cluster.eks_cluster_data[0].vpc_config[0].subnet_ids
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
                    createdBy = "kubelet-plugin-test-ci"
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
    } : null
}

module manage_eks_nodes_using_jfrog_credential_plugin_web_identity {
  enable_aws = var.enable_aws
  source = "./../terraform-module"
  region = var.region

  jfrog_credential_provider_binary_url = var.jfrog_credential_provider_binary_url
  artifactory_url  = var.artifactory_url

  aws_service_token_exchange = true

  artifactory_glob_pattern = var.artifactory_glob_pattern

  artifactory_user = var.artifactory_user_wi

  create_eks_node_groups = true

  authentication_method = "assume_role"

  iam_role_arn = var.enable_aws ? local.iam_role_arn : null

  wait_for_creation = var.enable_aws ? aws_iam_role.eks_node_role[0].arn : ""

  eks_node_group_configuration = var.enable_aws ? {
    node_role_arn                   = local.iam_role_arn
    cluster_name                    = !var.create_eks_cluster ? var.self_managed_eks_cluster.name : module.eks[0].cluster_name
    cluster_service_ipv4_cidr       = data.aws_eks_cluster.eks_cluster_data[0].kubernetes_network_config[0].service_ipv4_cidr
    subnet_ids                      = data.aws_eks_cluster.eks_cluster_data[0].vpc_config[0].subnet_ids
    node_groups = [
      {
        name            = "jfrog-credential-plugin-web-id-arm64"
        vpc_security_group_ids = var.create_eks_cluster ? [module.eks[0].node_security_group_id] : var.node_security_group_ids
        desired_size    = var.node_group_desired_size
        max_size        = var.node_group_max_size
        min_size        = var.node_group_min_size
        ami_type        = var.ami_type
        instance_types  = var.node_group_instance_types
        labels          = {
          createdBy = "kubelet-plugin-test-ci"
          nodeType = "web-identity"
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
  } : null
}

# Azure Module Calls
module create_azure_daemonset_with_plugin_enabled {
    enable_azure = var.enable_azure
    source = "./../terraform-module"
    
    jfrog_credential_provider_binary_url = var.jfrog_credential_provider_binary_url
    artifactory_url  = var.artifactory_url
    artifactory_glob_pattern = var.artifactory_glob_pattern
    artifactory_user = "azure-aks-user"

    # Azure only supports DaemonSet installation
    create_eks_node_groups = false
    jfrog_credential_plugin_daemonset_installation = true

    # Azure authentication configuration
    azure_envs = var.enable_azure ? {
        azure_app_client_id = local.azure_client_id
        azure_tenant_id = data.azuread_client_config.current[0].tenant_id
        azure_app_audience = "api://AzureADTokenExchange"
        azure_nodepool_client_id = data.azurerm_kubernetes_cluster.k8s[0].kubelet_identity[0].client_id
    } : null
    jfrog_oidc_provider_name = var.jfrog_oidc_provider_name

    wait_for_creation = var.enable_azure ? data.azurerm_kubernetes_cluster.k8s[0].kubelet_identity[0].client_id : ""

    kubernetes_auth_object = var.enable_azure ? {
        host                   = var.create_aks_cluster ? azurerm_kubernetes_cluster.k8s[0].kube_config[0].host : data.azurerm_kubernetes_cluster.k8s[0].kube_config[0].host
        cluster_ca_certificate = var.create_aks_cluster ? base64decode(azurerm_kubernetes_cluster.k8s[0].kube_config[0].cluster_ca_certificate) : base64decode(data.azurerm_kubernetes_cluster.k8s[0].kube_config[0].cluster_ca_certificate)
        token                  = ""  # Azure uses different auth mechanism
        client_certificate = var.create_aks_cluster ? base64decode(azurerm_kubernetes_cluster.k8s[0].kube_config[0].client_certificate) : base64decode(data.azurerm_kubernetes_cluster.k8s[0].kube_config[0].client_certificate)
        client_key = var.create_aks_cluster ? base64decode(azurerm_kubernetes_cluster.k8s[0].kube_config[0].client_key) : base64decode(data.azurerm_kubernetes_cluster.k8s[0].kube_config[0].client_key)
    } : {}

    daemonset_configuration = {
        jfrog_namespace = var.jfrog_namespace
        node_selector   = [{
            key: "azure-jfrog-test"
            value: "true"
        }]
        tolerations = []
    }
}

module create_daemonset_with_plugin_enabled {
    enable_aws = var.enable_aws
    source = "./../terraform-module"
    region = var.region

    jfrog_credential_provider_binary_url = var.jfrog_credential_provider_binary_url
    artifactory_url  = var.artifactory_url
    artifactory_glob_pattern = var.artifactory_glob_pattern

    artifactory_user = var.artifactory_user

    create_eks_node_groups = false
    jfrog_credential_plugin_daemonset_installation = var.enable_aws ? true : false

    authentication_method = "cognito_oidc"

    iam_role_arn = var.enable_aws ? local.iam_role_arn : null

    wait_for_creation = var.enable_aws ? aws_iam_role.eks_node_role[0].arn : ""

    kubernetes_auth_object  = var.enable_aws ? {
        host                   =  var.create_eks_cluster ? module.eks[0].cluster_endpoint : data.aws_eks_cluster.eks_cluster_data[0].endpoint
        cluster_ca_certificate = var.create_eks_cluster ? base64decode(module.eks[0].cluster_certificate_authority_data) : base64decode(data.aws_eks_cluster.eks_cluster_data[0].certificate_authority.0.data)
        token                  = data.aws_eks_cluster_auth.eks_cluster_auth[0].token
    } : {}

    # OIDC Configuration (Required when authentication_method = "cognito_oidc")
    jfrog_oidc_provider_name = var.jfrog_oidc_provider_name
    aws_cognito_user_pool_name = var.enable_aws ? aws_cognito_user_pool.jfrog_cognito_user_pool[0].name : null
    aws_cognito_user_pool_id = var.enable_aws ? aws_cognito_user_pool.jfrog_cognito_user_pool[0].id : null
    aws_cognito_user_pool_client_id = var.enable_aws ? aws_cognito_user_pool_client.jfrog_user_pool_client[0].id : null
    aws_cognito_resource_server_name = var.enable_aws ? aws_cognito_resource_server.jfrog_oidc_resource[0].name : null
    aws_cognito_user_pool_domain_name = var.enable_aws ? local.cognito_user_pool_domain_name : null
    aws_cognito_user_pool_secret_name = var.enable_aws ? aws_secretsmanager_secret.jfrog_oidc_integration_secret[0].name : null

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