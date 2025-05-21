data "aws_availability_zones" "available" {
    filter {
        name   = "opt-in-status"
        values = ["opt-in-not-required"]
    }
}

data "aws_eks_cluster" "self_managed_eks_cluster_data" {
    name = var.self_managed_eks_cluster["name"]
}

data "aws_eks_cluster_auth" "self_managed_eks_cluster_auth" {
  name = data.aws_eks_cluster.self_managed_eks_cluster_data.name
}


