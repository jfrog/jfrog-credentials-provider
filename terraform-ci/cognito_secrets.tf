resource "aws_cognito_user_pool" "jfrog_cognito_user_pool" {
    name = "jfrog-user-pool_${var.region}"
}

resource "aws_cognito_user_pool_client" "jfrog_user_pool_client" {
    name = "jfrog-user-pool_client"
    user_pool_id = aws_cognito_user_pool.jfrog_cognito_user_pool.id

    generate_secret = true

    callback_urls                        = ["https://example.com"]
    allowed_oauth_flows_user_pool_client = true
    allowed_oauth_flows                  = ["client_credentials"]
    allowed_oauth_scopes                 = ["${aws_cognito_resource_server.jfrog_oidc_resource.identifier}/read"]
    supported_identity_providers         = ["COGNITO"]
    access_token_validity                = 10
    id_token_validity                    = 10
    refresh_token_validity               = 30

    token_validity_units {
        refresh_token = "days"
        access_token = "minutes"
        id_token = "minutes"
    }
}

resource "aws_cognito_user_pool_domain" "jfrog_oidc_domain" {
    domain       = local.cognito_user_pool_domain_name
    user_pool_id = aws_cognito_user_pool.jfrog_cognito_user_pool.id
}

resource "aws_cognito_resource_server" "jfrog_oidc_resource" {
    identifier = aws_cognito_user_pool.jfrog_cognito_user_pool.id
    name       = "jfrog-oidc-resource"

    user_pool_id = aws_cognito_user_pool.jfrog_cognito_user_pool.id
    scope {
        scope_name        = "read"
        scope_description = "Read only"
    }
}

resource "aws_secretsmanager_secret" "jfrog_oidc_integration_secret" {
    name = "jfrog_oidc_integration_secret_${var.region}"

    recovery_window_in_days = 0
    force_overwrite_replica_secret = true
}

resource "aws_secretsmanager_secret_version" "jfrog_oidc_integration_secret_version" {
    secret_id     = aws_secretsmanager_secret.jfrog_oidc_integration_secret.id
    secret_string = local.jfrog_oidc_provider_secret
}