# resource "null_resource" "configure_artifactory_oidc" {
#   # Ensure this runs only after the Cognito resources are created
#   depends_on = [
#     aws_cognito_user_pool.jfrog_cognito_user_pool,
#     aws_cognito_user_pool_client.jfrog_user_pool_client
#   ]

#   # Triggers: Re-run this provisioner if any of these values change
#   triggers = {
#     artifactory_url                         = var.artifactory_url
#     jfrog_oidc_provider_name                = var.jfrog_oidc_provider_name
#     region                                  = var.region
#     cognito_user_pool_id                    = aws_cognito_user_pool.jfrog_cognito_user_pool.id
#     cognito_user_pool_client_id             = aws_cognito_user_pool_client.jfrog_user_pool_client.id
#     artifactory_oidc_identity_mapping_username = var.artifactory_oidc_identity_mapping_username
#     artifactory_aws_iam_role_arn           = module.daemonset_test_ng.iam_role_arn
#   }

#   provisioner "local-exec" {
#     command = <<-EOT
#       set -e
#       if [ -z "$JFROG_CI_ARTIFACTORY_API_TOKEN" ]; then
#         echo "Error: JFROG_CI_ARTIFACTORY_API_TOKEN environment variable is not set."
#         exit 1
#       fi

#       echo "Attempting to configure Artifactory OIDC provider: ${self.triggers.jfrog_oidc_provider_name}"

#       echo "Deleting existing OIDC provider (if any)..."
#       curl -sf -X DELETE "https://${self.triggers.artifactory_url}/access/api/v1/oidc/${self.triggers.jfrog_oidc_provider_name}" \
#           -H "Authorization: Bearer $JFROG_CI_ARTIFACTORY_API_TOKEN" || echo "OIDC provider ${self.triggers.jfrog_oidc_provider_name} not found or deletion failed, continuing..."

#       echo "Creating OIDC provider configuration in Artifactory..."
#       curl -sf -XPOST "https://${self.triggers.artifactory_url}/access/api/v1/oidc" \
#           -H "Content-Type: application/json" \
#           -H "Authorization: Bearer $JFROG_CI_ARTIFACTORY_API_TOKEN" \
#           -d '{
#             "name": "${self.triggers.jfrog_oidc_provider_name}",
#             "issuer_url": "https://cognito-idp.${self.triggers.region}.amazonaws.com/${self.triggers.cognito_user_pool_id}/",
#             "description": "OIDC with AWS (Managed by Terraform)",
#             "provider_type": "Generic OpenID Connect",
#             "token_issuer": "https://cognito-idp.${self.triggers.region}.amazonaws.com/${self.triggers.cognito_user_pool_id}",
#             "use_default_proxy": false
#           }'

#       echo "Configuring Artifactory OIDC provider identity mapping..."
#       curl -sf -XPOST "https://${self.triggers.artifactory_url}/access/api/v1/oidc/${self.triggers.jfrog_oidc_provider_name}/identity_mappings" \
#           -H "Content-Type: application/json" \
#           -H "Authorization: Bearer $JFROG_CI_ARTIFACTORY_API_TOKEN" \
#           -d '{
#             "name": "${self.triggers.jfrog_oidc_provider_name}",
#             "description": "OIDC identity mapping (Managed by Terraform)",
#             "claims": {
#               "client_id": "${self.triggers.cognito_user_pool_client_id}",
#               "iss": "https://cognito-idp.${self.triggers.region}.amazonaws.com/${self.triggers.cognito_user_pool_id}"
#             },
#             "token_spec": {
#               "username": "${self.triggers.artifactory_oidc_identity_mapping_username}",
#               "scope": "applied-permissions/user",
#               "audience": "*@*",
#               "expires_in": 330
#             },
#             "priority": 1
#           }'
#       echo "Artifactory OIDC configuration complete."

#       curl -sf -XPUT "https://${self.triggers.artifactory_url}/access/api/v1/aws/iam_role" \
#           -H "Content-Type: application/json" \
#           -H "Authorization: Bearer $JFROG_CI_ARTIFACTORY_API_TOKEN" \
#           -d '{
#             "username": "${self.triggers.artifactory_oidc_identity_mapping_username}",
#             "iam_role": "${self.triggers.artifactory_aws_iam_role_arn}"
#           }'
#       echo "Artifactory AWS IAM role binding configuration complete."
#     EOT
#     interpreter = ["bash", "-c"]
#   }
# }