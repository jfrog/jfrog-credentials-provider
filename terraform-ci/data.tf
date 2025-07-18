data "aws_availability_zones" "available" {
    filter {
        name   = "opt-in-status"
        values = ["opt-in-not-required"]
    }
}

data "aws_eks_cluster" "eks_cluster_data" {
    name = !var.create_eks_cluster ? var.self_managed_eks_cluster["name"] : module.eks[0].cluster_name

    depends_on = [module.eks]
}

data "aws_eks_cluster_auth" "eks_cluster_auth" {
   name = data.aws_eks_cluster.eks_cluster_data.name
}

data "aws_caller_identity" "current" {}