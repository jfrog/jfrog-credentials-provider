locals {
   oidc_config = var.enable_azure ? {
    issuer_url = "https://login.microsoftonline.com/${var.azure_envs.azure_tenant_id}/v2.0" 
    provider_type = "Azure"
    token_issuer = "https://login.microsoftonline.com/${var.azure_envs.azure_tenant_id}/v2.0"
    claims = {
      aud = var.azure_envs.azure_app_client_id
      iss = "https://login.microsoftonline.com/${var.azure_envs.azure_tenant_id}/v2.0"
    }
   } : var.enable_aws && var.authentication_method == "cognito_oidc"? {
    issuer_url = "https://cognito-idp.${var.region}.amazonaws.com/${var.aws_cognito_user_pool_id}/"
    provider_type = "Generic OpenID Connect"
    token_issuer = "https://cognito-idp.${var.region}.amazonaws.com/${var.aws_cognito_user_pool_id}/"
    claims = {
      client_id = var.aws_cognito_user_pool_client_id
      iss = "https://cognito-idp.${var.region}.amazonaws.com/${var.aws_cognito_user_pool_id}"
    }
   } : {
    issuer_url = ""
    provider_type = ""
    token_issuer = ""
    claims = {}
   }
}

resource "null_resource" "configure_artifactory_oidc" {
  count = var.enable_aws || var.enable_azure ? 1 : 0
  triggers = {
    artifactory_url                             = var.artifactory_url
    jfrog_oidc_provider_name                    = var.jfrog_oidc_provider_name
    artifactory_user                            = var.artifactory_user
    artifactory_aws_iam_role_arn                = var.enable_aws ? var.iam_role_arn : ""
    authentication_method                       = var.enable_aws ? var.authentication_method : ""
    cloud_provider                              = var.enable_aws ? "aws" : var.enable_azure ? "azure" : "none"

    # oidc config
    issuer_url = local.oidc_config.issuer_url
    provider_type = local.oidc_config.provider_type
    token_issuer = local.oidc_config.token_issuer
    claims = jsonencode(local.oidc_config.claims)
  }

  provisioner "local-exec" {
    command = <<-EOT
      set -e
      echo "Recieved triggers for Artifactory OIDC configuration with cloud provider: ${self.triggers.cloud_provider}"
      if [ -z "$ARTIFACTORY_TOKEN" ]; then
        echo "Error: ARTIFACTORY_TOKEN environment variable is not set."
        exit 1
      fi

      if [[ "${self.triggers.cloud_provider}" = "aws" && "${self.triggers.authentication_method}" = "assume_role" ]]; then
      echo "Delete existing AWS IAM role binding (if any)..."
      curl  -X DELETE "https://${self.triggers.artifactory_url}/access/api/v1/aws/iam_role/${self.triggers.artifactory_user}" \
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

      if [[ "${self.triggers.cloud_provider}" = "azure" || ("${self.triggers.cloud_provider}" = "aws" && "${self.triggers.authentication_method}" = "cognito_oidc") ]]; then

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
            "issuer_url": "${self.triggers.issuer_url}",
            "description": "OIDC (Managed by Terraform)",
            "provider_type": "${self.triggers.provider_type}",
            "token_issuer": "${self.triggers.token_issuer}",
            "use_default_proxy": false
          }'

      echo "Configuring Artifactory OIDC provider identity mapping..."
      curl -XPOST "https://${self.triggers.artifactory_url}/access/api/v1/oidc/${self.triggers.jfrog_oidc_provider_name}/identity_mappings" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $ARTIFACTORY_TOKEN" \
          -d '{
            "name": "${self.triggers.jfrog_oidc_provider_name}",
            "description": "OIDC identity mapping (Managed by Terraform)",
            "claims": ${self.triggers.claims},
            "token_spec": {
              "username": "${self.triggers.artifactory_user}",
              "scope": "applied-permissions/user",
              "audience": "*@*",
              "expires_in": 14400
            },
            "priority": 1
          }'
      echo "Artifactory OIDC configuration complete."
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
      curl  -X DELETE "https://${self.triggers.artifactory_url}/access/api/v1/aws/iam_role/${self.triggers.artifactory_user}" \
          -H "Authorization: Bearer $ARTIFACTORY_TOKEN" || echo "AWS IAM role binding not found or deletion failed"
    EOT
    interpreter = ["bash", "-c"]
    
  }
}
