resource "local_file" "jfrog_provider_oidc" {
    count = var.authentication_method == "cognito_oidc" && var.cloud_provider == "aws" ? 1 : 0
    content  = <<-EOT
        {
        "name": "jfrog-credential-provider",
        "matchImages": [
            "*.jfrog.io"
        ],
        "defaultCacheDuration": "4h",
        "apiVersion": "credentialprovider.kubelet.k8s.io/v1",
        "env": [
            {
            "name": "artifactory_url",
            "value": "${var.artifactory_url}"
            },
            {
            "name": "jfrog_oidc_provider_name",
            "value": "${var.jfrog_oidc_provider_name}"
            },
            {
            "name": "aws_auth_method",
            "value": "${var.authentication_method}"
            },
            {
            "name": "user_pool_name",
            "value": "${var.aws_cognito_user_pool_name}"
            },
            {
            "name": "resource_server_name",
            "value": "${var.aws_cognito_resource_server_name}"
            },
            {
            "name": "user_pool_resource_domain_name",
            "value": "${var.aws_cognito_user_pool_domain_name}"
            },
            {
            "name": "user_pool_resource_scope",
            "value": "read"
            },
            {
            "name": "secretTTL",
            "value": "5h"
            },
            {
            "name": "secret_name",
            "value": "${var.aws_cognito_user_pool_secret_name}"
            },
            {
            "name": "aws_role_name",
            "value": "${local.iam_role_name}"
          }
        ]
        }

        EOT
    filename = "${path.module}/jfrog/jfrog-provider.json"
}

resource "local_file" "jfrog_provider_assume_role"  {
    count = var.authentication_method == "assume_role" && var.cloud_provider == "aws" ? 1 : 0
    content = <<-EOT
    {
      "name": "jfrog-credential-provider",
      "matchImages": [
        "*.jfrog.io"
      ],
      "defaultCacheDuration": "4h",
      "apiVersion": "credentialprovider.kubelet.k8s.io/v1",
      "env": [
        {
          "name": "artifactory_url",
          "value": "${var.artifactory_url}"
        },
        {
          "name": "aws_auth_method",
          "value": "assume_role"
        },
        {
          "name": "aws_role_name",
          "value": "${local.iam_role_name}"
        },
        {
        "name": "secretTTL",
        "value": "5h"
        }
      ]
    } 

    EOT
    filename = "${path.module}/jfrog/jfrog-provider.json"
}

resource "local_file" "jfrog_provider_azure" {
    count = var.cloud_provider == "azure" ? 1 : 0
    content = <<-EOT
    providers:
    - name: jfrog-credential-provider
      matchImages:
        - "*.jfrog.io"
      apiVersion: credentialprovider.kubelet.k8s.io/v1
      defaultCacheDuration: "1h"
      env:
      - name: artifactory_url
        value: "${var.artifactory_url}"
      - name: azure_app_client_id
        value: "${var.azure_envs.azure_app_client_id}"
      - name: azure_tenant_id
        value: "${var.azure_envs.azure_tenant_id}"
      - name: azure_app_audience
        value: "${var.azure_envs.azure_app_audience}"
      - name: jfrog_oidc_provider_name
        value: "${var.jfrog_oidc_provider_name}"
      - name: azure_nodepool_client_id
        value: "${var.azure_envs.azure_nodepool_client_id}"
    EOT
    filename = "${path.module}/jfrog/jfrog-provider.yaml"
}

locals {
    jfrog_provider_aws_config_content = var.cloud_provider == "aws" ? (
      var.authentication_method == "cognito_oidc" ? (
        length(local_file.jfrog_provider_oidc) > 0 ? local_file.jfrog_provider_oidc[0].content : ""
    ) : (
        var.authentication_method == "assume_role" ? (
        length(local_file.jfrog_provider_assume_role) > 0 ? local_file.jfrog_provider_assume_role[0].content : ""
        ) : ""
    ) ) : ""

    # Azure only supports the DaemonSet installation method
    jfrog_provider_azure_config_content = var.cloud_provider == "azure" ? (
        length(local_file.jfrog_provider_azure) > 0 ? local_file.jfrog_provider_azure[0].content : ""
    ) : ""
}

module "eks_managed_node_group" {
    for_each = var.cloud_provider == "aws" ? (var.create_eks_node_groups ? { for ng in var.eks_node_group_configuration.node_groups : ng.name => ng } : {}) : {}
    source   = "terraform-aws-modules/eks/aws//modules/eks-managed-node-group"
    version  = "~> 21.0.0"

    cluster_name                = var.eks_node_group_configuration.cluster_name
    use_latest_ami_release_version = false
    name                        = each.value.name
    cluster_service_cidr       = var.eks_node_group_configuration.cluster_service_ipv4_cidr
    subnet_ids                  = var.eks_node_group_configuration.subnet_ids
    desired_size                = each.value.desired_size
    max_size                    = each.value.max_size
    min_size                    = each.value.min_size
    ami_type                    = each.value.ami_type
    instance_types              = each.value.instance_types
    create_iam_role            = false
    iam_role_arn                = var.eks_node_group_configuration.node_role_arn
    vpc_security_group_ids      = each.value.security_group_ids != null ? each.value.security_group_ids : []

    pre_bootstrap_user_data = <<-EOF
        echo '${local.jfrog_provider_aws_config_content}' > /etc/eks/image-credential-provider/jfrog-provider.json

        export JFROG_CREDENTIAL_PROVIDER_BINARY_URL="${var.jfrog_credential_provider_binary_url}"
        export ARTIFACTORY_URL="${var.artifactory_url}"
        ${file("${path.module}/jfrog/bootstrap.sh")}
        EOF
        
    cloudinit_pre_nodeadm  = [
    {
        content = <<-EOF
        #!bin/bash
        echo '${local.jfrog_provider_aws_config_content}' > /etc/eks/image-credential-provider/jfrog-provider.json

        export JFROG_CREDENTIAL_PROVIDER_BINARY_URL="${var.jfrog_credential_provider_binary_url}"
        export ARTIFACTORY_URL="${var.artifactory_url}"
        ${file("${path.module}/jfrog/bootstrap.sh")}
        EOF
    }
    ]
    labels = each.value.labels != null ? each.value.labels : {}
    taints = each.value.taints != null ? each.value.taints : {}
}