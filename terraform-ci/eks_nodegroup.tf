
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
        ami_type = "AL2023_ARM_64_STANDARD"
        create_iam_role          = false
        iam_role_arn             = aws_iam_role.eks_node_role.arn
    }

    eks_managed_node_groups = {
        main = {
            name = "eks-system-node-group"

            instance_types = ["t4g.medium"]

            min_size     = 1
            max_size     = 3
            desired_size = 2
        }
    }
}


module "daemonset_test_ng" {
  source = "terraform-aws-modules/eks/aws//modules/eks-managed-node-group"

  name         = "kubelet-plugin-daemonset-test-ng"
  cluster_name = var.create_eks_cluster ? module.eks[0].cluster_name : var.self_managed_eks_cluster.name
  subnet_ids = data.aws_eks_cluster.eks_cluster_data.vpc_config[0].subnet_ids
  cluster_service_ipv4_cidr = data.aws_eks_cluster.eks_cluster_data.kubernetes_network_config[0].service_ipv4_cidr

  instance_types = var.daemonset_node_group_instance_types
  ami_type       = var.ami_type
  min_size       = var.daemonset_node_group_min_size
  max_size       = var.daemonset_node_group_max_size
  desired_size   = var.daemonset_node_group_desired_size

  create_iam_role           = false
  iam_role_arn              = aws_iam_role.eks_node_role.arn

    labels = {
        onlyForDaemonset = "true"
    }
    taints = [
        {
          key    = "forDaemonset"
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
