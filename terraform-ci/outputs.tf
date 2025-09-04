# AWS Outputs
output "busybox_pod_status" {
  description = "The full status of the AWS busybox pod"
  value       = var.enable_aws ? kubernetes_pod_v1.busybox_pod[0] : null
}

output "busybox_ds_pod_status" {
  description = "The full status of the AWS busybox DaemonSet pod"
  value       = var.enable_aws ? kubernetes_pod_v1.busybox_pod_ds[0] : null
}

output "aws_cluster_endpoint" {
  description = "AWS EKS cluster endpoint"
  value       = var.enable_aws ? (var.create_eks_cluster ? module.eks[0].cluster_endpoint : data.aws_eks_cluster.eks_cluster_data[0].endpoint) : null
}

# Azure Outputs
output "azure_busybox_ds_pod_status" {
  description = "The full status of the Azure busybox DaemonSet pod"
  value       = var.enable_azure ? kubernetes_pod_v1.azure_busybox_pod_ds[0] : null
}