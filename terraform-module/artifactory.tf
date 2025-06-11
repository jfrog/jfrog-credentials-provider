
resource "null_resource" "configure_artifactory_oidc" {
  triggers = {
    artifactory_url                             = var.artifactory_url
    jfrog_oidc_provider_name                    = var.jfrog_oidc_provider_name
    region                                      = var.region
    # TODO Change this to the actual Cognito User Pool ID
    cognito_user_pool_id                       = var.authentication_method == "cognito_oidc" ? var.aws_cognito_user_pool_id : ""
    cognito_user_pool_client_id                = var.authentication_method == "cognito_oidc" ? var.aws_cognito_user_pool_client_id : ""
    artifactory_user                           = var.artifactory_user
    artifactory_aws_iam_role_arn                = var.iam_role_arn
    authentication_method                       = var.authentication_method
  }

  provisioner "local-exec" {
    command = <<-EOT
      set -e
      echo "Recieved triggers: ${self.triggers.artifactory_aws_iam_role_arn}"
      
      if [ -z "$ARTIFACTORY_TOKEN" ]; then
        echo "Error: ARTIFACTORY_TOKEN environment variable is not set."
        exit 1
      fi

      if [ "${self.triggers.authentication_method}" = "cognito_oidc" ]; then
      echo "Attempting to configure Artifactory OIDC provider: ${self.triggers.jfrog_oidc_provider_name}"

      echo "Deleting existing OIDC provider (if any)..."
      curl  -X DELETE "https://${self.triggers.artifactory_url}/access/api/v1/oidc/${self.triggers.jfrog_oidc_provider_name}" \
          -H "Authorization: Bearer $ARTIFACTORY_TOKEN" || echo "OIDC provider ${self.triggers.jfrog_oidc_provider_name} not found or deletion failed, continuing..."

      echo "Creating OIDC provider configuration in Artifactory..."
      curl  -XPOST "https://${self.triggers.artifactory_url}/access/api/v1/oidc" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $ARTIFACTORY_TOKEN" \
          -d '{
            "name": "${self.triggers.jfrog_oidc_provider_name}",
            "issuer_url": "https://cognito-idp.${self.triggers.region}.amazonaws.com/${self.triggers.cognito_user_pool_id}/",
            "description": "OIDC with AWS (Managed by Terraform)",
            "provider_type": "Generic OpenID Connect",
            "token_issuer": "https://cognito-idp.${self.triggers.region}.amazonaws.com/${self.triggers.cognito_user_pool_id}",
            "use_default_proxy": false
          }'

      echo "Configuring Artifactory OIDC provider identity mapping..."
      curl -XPOST "https://${self.triggers.artifactory_url}/access/api/v1/oidc/${self.triggers.jfrog_oidc_provider_name}/identity_mappings" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $ARTIFACTORY_TOKEN" \
          -d '{
            "name": "${self.triggers.jfrog_oidc_provider_name}",
            "description": "OIDC identity mapping (Managed by Terraform)",
            "claims": {
              "client_id": "${self.triggers.cognito_user_pool_client_id}",
              "iss": "https://cognito-idp.${self.triggers.region}.amazonaws.com/${self.triggers.cognito_user_pool_id}"
            },
            "token_spec": {
              "username": "${self.triggers.artifactory_user}",
              "scope": "applied-permissions/user",
              "audience": "*@*",
              "expires_in": 330
            },
            "priority": 1
          }'
      echo "Artifactory OIDC configuration complete."
      else
        echo "Using assume_role authentication method. No OIDC provider configuration needed."
      fi

      if [ "${self.triggers.authentication_method}" = "assume_role" ]; then
      echo "Delete existing AWS IAM role binding (if any)..."
      curl  -X DELETE "https://${self.triggers.artifactory_url}/access/api/v1/aws/iam_role" \
          -H "Authorization: Bearer $ARTIFACTORY_TOKEN" || echo "AWS IAM role binding not found or deletion failed, continuing..."

      echo "Configuring Artifactory AWS IAM role binding..."
      curl  -XPUT "https://${self.triggers.artifactory_url}/access/api/v1/aws/iam_role" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $ARTIFACTORY_TOKEN" \
          -d '{
            "username": "${self.triggers.artifactory_user}",
            "iam_role": "${self.triggers.artifactory_aws_iam_role_arn}"
          }'
      echo "Artifactory AWS IAM role binding configuration complete."
      fi
    EOT
    interpreter = ["bash", "-c"]
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      if [ -z "$ARTIFACTORY_TOKEN" ]; then
        echo "Error: ARTIFACTORY_TOKEN environment variable is not set."
        exit 1
      fi

      echo "Deleting existing OIDC provider (if any)..."
      curl  -X DELETE "https://${self.triggers.artifactory_url}/access/api/v1/oidc/${self.triggers.jfrog_oidc_provider_name}" \
          -H "Authorization: Bearer $ARTIFACTORY_TOKEN" || echo "OIDC provider ${self.triggers.jfrog_oidc_provider_name} not found or deletion failed, continuing..."

      echo "Deleting existing AWS IAM role binding (if any)..."
      curl  -X DELETE "https://${self.triggers.artifactory_url}/access/api/v1/aws/iam_role" \
          -H "Authorization: Bearer $ARTIFACTORY_TOKEN" || echo "AWS IAM role binding not found or deletion failed"
    EOT
    interpreter = ["bash", "-c"]
    
  }
}
