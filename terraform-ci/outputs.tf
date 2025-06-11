output "busybox_pod_status" {
  description = "The full status of the busybox pod"
  value       = kubernetes_pod_v1.busybox_pod
}

output "busybox_ds_pod_status" {
  description = "The full status of the busybox pod"
  value       = kubernetes_pod_v1.busybox_pod_ds
}