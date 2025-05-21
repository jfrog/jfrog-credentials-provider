# Output the command to configure kubectl config to the newly created EKS cluster
output "_1_setting_cluster_kubectl_context" {
  description = "Connect kubectl to Kubernetes Cluster"
  value       = var.create_eks_cluster ?  "aws eks --region ${var.region} update-kubeconfig --name ${module.eks[0].cluster_name}" : "Used self managed EKS"
}

output "_2_setup_artifactory_oidc" {
  value = <<-EOT
  # Delete the existing OIDC provider if exists
  curl -X DELETE "https://${var.artifactory_url}/access/api/v1/oidc/${var.jfrog_oidc_provider_name}" \
      -H "Authorization: Bearer $${ARTIFACTORY_ADMIN_ACCESS_TOKEN}"

  # Create an OIDC provider configuration in Artifactory
  curl -XPOST "https://${var.artifactory_url}/access/api/v1/oidc" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $${ARTIFACTORY_ADMIN_ACCESS_TOKEN}" \
      -d '{
        "name": "${var.jfrog_oidc_provider_name}",
        "issuer_url": "https://cognito-idp.${var.region}.amazonaws.com/${aws_cognito_user_pool.jfrog_cognito_user_pool.id}/",
        "description": "OIDC with AWS",
        "provider_type": "Generic OpenID Connect",
        "token_issuer": "https://cognito-idp.${var.region}.amazonaws.com/${aws_cognito_user_pool.jfrog_cognito_user_pool.id}",
        "use_default_proxy": false
      }'

EOT
}

output "_3_setup_identity_mapping_oidc_integration" {
  value = <<-EOT
  # Run the following command to configure the Artifactory OIDC provider with identity mapping
  curl -XPOST "https://${var.artifactory_url}/access/api/v1/oidc/${var.jfrog_oidc_provider_name}/identity_mappings" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $${ARTIFACTORY_ADMIN_ACCESS_TOKEN}" \
      -d '{
        "name": "${var.jfrog_oidc_provider_name}",
        "description": "OIDC identity mapping",
        "claims": {
          "client_id": "${aws_cognito_user_pool_client.jfrog_user_pool_client.id}",
          "iss": "https://cognito-idp.${var.region}.amazonaws.com/${aws_cognito_user_pool.jfrog_cognito_user_pool.id}"
        },
        "token_spec": {
          "username": "${var.artifactory_oidc_identity_mapping_username}",
          "scope": "applied-permissions/user",
          "audience": "*@*",
          "expires_in": 330
        },
        "priority": 1
      }'

EOT
}

output "busybox_pod_status" {
  description = "The full status of the busybox pod"
  value       = kubernetes_pod_v1.busybox_pod
}

output "busybox_ds_pod_status" {
  description = "The full status of the busybox pod"
  value       = kubernetes_pod_v1.busybox_pod_ds
}