# This file is used to create an
# AWS VPC and EKS cluster with a managed node group
# It also creates a gp3 storage class and makes it the default

# data "aws_caller_identity" "current" {}


resource "aws_security_group_rule" "allow_management_from_my_ip" {
    count = var.create_eks_cluster ? 1 : 0
    type              = "ingress"
    from_port         = 0
    to_port           = 65535
    protocol          = "-1"
    cidr_blocks       = var.cluster_public_access_cidrs
    security_group_id = module.eks[0].cluster_security_group_id
    description       = "Allow all traffic from my public IP for management"
}


module "eks" {
    count = var.create_eks_cluster ? 1 : 0

    source  = "terraform-aws-modules/eks/aws"

    cluster_name    = local.cluster_name
    cluster_version = var.cluster_version

    enable_cluster_creator_admin_permissions = true
    cluster_endpoint_public_access           = true
    cluster_endpoint_public_access_cidrs     = var.cluster_public_access_cidrs

    cluster_addons = {
        aws-ebs-csi-driver = {
            most_recent = true
            service_account_role_arn = module.ebs_csi_irsa_role[0].iam_role_arn
        }
    }

    vpc_id     = module.vpc[0].vpc_id
    subnet_ids = module.vpc[0].private_subnets

    eks_managed_node_group_defaults = {
        ami_type = var.ami_type
        create_iam_role          = true
        iam_role_name            = local.node_role_name
        iam_role_use_name_prefix = false
        iam_role_description     = "EKS nodes role with a custom policy to allow Artifactory to get caller identity"

        iam_role_additional_policies = {
            AmazonEBSCSIDriverPolicy = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
            AmazonEC2ContainerRegistryReadOnly = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
            AmazonEKS_CNI_Policy = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
            AmazonEKSWorkerNodePolicy = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
            JFrogAllowGetSecretValuePolicy = aws_iam_policy.get_secret_value.arn
            JFrogAllowGetUserPoolPolicy = aws_iam_policy.get_user_pool.arn
        }

        pre_bootstrap_user_data = <<-EOF
        echo '${local_file.jfrog_provider.content}' > /etc/eks/image-credential-provider/jfrog-credential-config.json

        export JFROG_CREDENTIAL_PROVIDER_BINARY_URL="${var.jfrog_credential_provider_binary_url}"
        export ARTIFACTORY_URL="${var.artifactory_url}"
        ${file("${path.module}/jfrog/bootstrap.sh")}
        EOF
    }

    eks_managed_node_groups = {
        main = {
            name = "eks-node-group"

            instance_types = var.oidc_node_group_instance_types

            min_size     = var.oidc_node_group_min_size
            max_size     = var.oidc_node_group_max_size
            desired_size = var.oidc_node_group_desired_size
            labels = {
                  createdBy = "kubelet-plugin-test-ci"
                  nodeType = "cogito-oidc"
              }
              taints = [
                  {
                    key    = "jfrog-kubelet-oidc-ng"
                    value  = "true"
                    effect = "NO_SCHEDULE"
                  }
                ]
        }
    }
}

# Policies to fetch the user pool and secret values


# Managed node group using module
module "eks_managed_node_group" {
    count = var.create_eks_cluster ? 0 : 1
    source = "terraform-aws-modules/eks/aws//modules/eks-managed-node-group"
    cluster_name = var.self_managed_eks_cluster.name
    name = "kubelet-test-ng"
    cluster_service_ipv4_cidr = data.aws_eks_cluster.self_managed_eks_cluster_data.kubernetes_network_config[0].service_ipv4_cidr
    subnet_ids = data.aws_eks_cluster.self_managed_eks_cluster_data.vpc_config[0].subnet_ids
    desired_size = var.oidc_node_group_desired_size
    max_size = var.oidc_node_group_max_size
    min_size = var.oidc_node_group_min_size
    ami_type = var.ami_type
    instance_types = var.oidc_node_group_instance_types
    create_iam_role          = true
    iam_role_name            = local.node_role_name
    iam_role_use_name_prefix = false
    iam_role_description     = "EKS nodes role with a custom policy to allow Artifactory to get caller identity"

    iam_role_additional_policies = {
        AmazonEBSCSIDriverPolicy = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
        AmazonEC2ContainerRegistryReadOnly = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
        AmazonEKS_CNI_Policy = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
        AmazonEKSWorkerNodePolicy = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
        JFrogAllowGetSecretValuePolicy = aws_iam_policy.get_secret_value.arn
        JFrogAllowGetUserPoolPolicy = aws_iam_policy.get_user_pool.arn
    }

    pre_bootstrap_user_data = <<-EOF
    echo '${local_file.jfrog_provider.content}' > /etc/eks/image-credential-provider/jfrog-credential-config.json

    export JFROG_CREDENTIAL_PROVIDER_BINARY_URL="${var.jfrog_credential_provider_binary_url}"
    export ARTIFACTORY_URL="${var.artifactory_url}"
    ${file("${path.module}/jfrog/bootstrap.sh")}
    EOF

    labels = {
        createdBy = "kubelet-plugin-test-ci"
        nodeType = "cogito-oidc"
    }
    taints = [
        {
          key    = "jfrog-kubelet-oidc-ng"
          value  = "true"
          effect = "NO_SCHEDULE"
        }
      ]
}

module "daemonset_test_ng" {
  source = "terraform-aws-modules/eks/aws//modules/eks-managed-node-group"

  name         = "kubelet-plugin-daemonset-test-ng"
  cluster_name = var.create_eks_cluster ? module.eks[0].cluster_name : var.self_managed_eks_cluster.name
  subnet_ids = var.create_eks_cluster ? module.vpc[0].private_subnets : data.aws_eks_cluster.self_managed_eks_cluster_data.vpc_config[0].subnet_ids
  cluster_service_ipv4_cidr = var.create_eks_cluster ? module.eks[0].cluster_service_ipv4_cidr : data.aws_eks_cluster.self_managed_eks_cluster_data.kubernetes_network_config[0].service_ipv4_cidr

  instance_types = var.daemonset_node_group_instance_types
  ami_type       = var.ami_type
  min_size       = var.daemonset_node_group_min_size
  max_size       = var.daemonset_node_group_max_size
  desired_size   = var.daemonset_node_group_desired_size

  create_iam_role          = true
  iam_role_name            = local.node_ds_role_name
  iam_role_use_name_prefix = false
  iam_role_description     = "IAM role for the Daemonset Test EKS node group"

  # Standard IAM policies for a general-purpose node group
  iam_role_additional_policies = {
        AmazonEBSCSIDriverPolicy = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
        AmazonEC2ContainerRegistryReadOnly = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
        AmazonEKS_CNI_Policy = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
        AmazonEKSWorkerNodePolicy = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  }

    labels = {
        createdBy = "kubelet-plugin-test-ci"
        nodeType = "daemonset"
    }
    taints = [
        {
          key    = "jfrog-kubelet-daemonset-ng"
          value  = "true"
          effect = "NO_SCHEDULE"
        }
      ]

}


module "ebs_csi_irsa_role" {
    count = var.create_eks_cluster ? 1 : 0
    source = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"

    role_name             = "ebs-csi-role-${local.cluster_name}-${var.region}"
    attach_ebs_csi_policy = true

    oidc_providers = {
        ex = {
            provider_arn               = module.eks[0].oidc_provider_arn
            namespace_service_accounts = ["kube-system:ebs-csi-controller-sa"]
        }
    }
}


# Prepare the jfrog_provider.json file
resource "local_file" "jfrog_provider" {
    content  = <<-EOT
    {
      "name": "jfrog-credential-provider",
      "matchImages": [
        "*.jfrog.io"
      ],
      "defaultCacheDuration": "5h",
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
          "value": "${aws_cognito_user_pool.jfrog_cognito_user_pool.name}"
        },
        {
          "name": "resource_server_name",
          "value": "${aws_cognito_resource_server.jfrog_oidc_resource.name}"
        },
        {
          "name": "user_pool_resource_domain_name",
          "value": "${local.cognito_user_pool_domain_name}"
        },
        {
          "name": "user_pool_resource_scope",
          "value": "read"
        },
        {
          "name": "secret_name",
          "value": "${aws_secretsmanager_secret.jfrog_oidc_integration_secret.name}"
        },
        {
          "name": "aws_role_name",
          "value": "${local.node_role_name}"
        }
      ]
    }

    EOT
    filename = "${path.module}/jfrog/jfrog_provider_generated.json"
}
