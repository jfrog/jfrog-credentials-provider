resource "aws_iam_role" "eks_node_role" {
  name  = local.node_role_name

  assume_role_policy = jsonencode({
    Version   = "2012-10-17",
    Statement = [
      {
        Action    = "sts:AssumeRole",
        Effect    = "Allow",
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  description = "EKS nodes role with a custom policy to allow Artifactory to get caller identity"
}

resource "aws_iam_role_policy_attachment" "eks_node_AmazonEKSWorkerNodePolicy" {
  role       = aws_iam_role.eks_node_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
}

resource "aws_iam_role_policy_attachment" "eks_node_AmazonEC2ContainerRegistryReadOnly" {
  role       = aws_iam_role.eks_node_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

resource "aws_iam_role_policy_attachment" "eks_node_AmazonEKS_CNI_Policy" {
  role       = aws_iam_role.eks_node_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
}

resource "aws_iam_role_policy_attachment" "eks_node_AmazonEBSCSIDriverPolicy" {
  role       = aws_iam_role.eks_node_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
}

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

resource "aws_iam_role_policy_attachment" "eks_node_JFrogAllowGetSecretValuePolicy" {
  role       = aws_iam_role.eks_node_role.name
  policy_arn = aws_iam_policy.get_secret_value.arn
}

resource "aws_iam_role_policy_attachment" "eks_node_JFrogAllowGetUserPoolPolicy" {
  role       = aws_iam_role.eks_node_role.name
  policy_arn = aws_iam_policy.get_user_pool.arn
}
