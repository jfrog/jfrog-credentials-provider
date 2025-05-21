resource "aws_iam_policy" "get_secret_value" {
    name        = "JFrogAllowGetSecretValuePolicy"
    description = "Allow EKS node to read the secret value"

    policy = jsonencode({
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "secretsmanager:GetSecretValue"
                ],
                "Resource": [
                    aws_secretsmanager_secret_version.jfrog_oidc_integration_secret_version.arn
                ]
            }
        ]
    })
}

resource "aws_iam_policy" "get_user_pool" {
    name        = "JFrogAllowGetUserPoolPolicy"
    description = "Allow EKS node to read the user pool details"

    policy = jsonencode({
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "cognito-idp:ListUserPools",
                    "cognito-idp:ListResourceServers"
                ],
                "Resource": [
                    "*"
                ]
            },
            {
                "Effect": "Allow",
                "Action": [
                    "cognito-idp:DescribeUserPool"
                ],
                "Resource": [
                    aws_cognito_user_pool.jfrog_cognito_user_pool.arn
                ]
            }
        ]
    })
}