resource "aws_iam_role" "eks_node_role" {
  count = var.enable_aws ? 1 : 0
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
  
  # Ignore changes to assume_role_policy initially to avoid cycle
  lifecycle {
    ignore_changes = [assume_role_policy]
  }
}

# Update the role's assume_role_policy after EKS cluster is created
resource "null_resource" "update_eks_node_role_web_identity" {
  for_each = var.enable_aws ? { update = true } : {}
  
  triggers = {
    eks_oidc_id = try(local.eks_oidc_id, "")
    role_name   = aws_iam_role.eks_node_role[0].name
    account_id  = data.aws_caller_identity.current[0].account_id
  }

  provisioner "local-exec" {
    command = <<-EOT
      # Only update if eks_oidc_id is available
      if [ -n "${try(local.eks_oidc_id, "")}" ]; then
        aws iam update-assume-role-policy \
          --role-name ${aws_iam_role.eks_node_role[0].name} \
          --policy-document '{
            "Version": "2012-10-17",
            "Statement": [
              {
                "Action": "sts:AssumeRole",
                "Effect": "Allow",
                "Principal": {
                  "Service": "ec2.amazonaws.com"
                }
              },
              {
                "Effect": "Allow",
                "Principal": {
                  "Federated": "arn:aws:iam::${data.aws_caller_identity.current[0].account_id}:oidc-provider/oidc.eks.${var.region}.amazonaws.com/id/${local.eks_oidc_id}"
                },
                "Action": "sts:AssumeRoleWithWebIdentity",
                "Condition": {
                  "StringEquals": {
                    "oidc.eks.${var.region}.amazonaws.com/id/${local.eks_oidc_id}:sub": "system:serviceaccount:${var.jfrog_namespace}:busybox-sa-wi",
                    "oidc.eks.${var.region}.amazonaws.com/id/${local.eks_oidc_id}:aud": "sts.amazonaws.com"
                  }
                }
              }
            ]
          }'
      fi
    EOT
  }

  depends_on = [
    module.eks,
    aws_iam_role.eks_node_role
  ]
}

resource "aws_iam_role_policy_attachment" "eks_node_AmazonEKSWorkerNodePolicy" {
  count = var.enable_aws ? 1 : 0
  role       = aws_iam_role.eks_node_role[0].name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
}

resource "aws_iam_role_policy_attachment" "eks_node_AmazonEC2ContainerRegistryReadOnly" {
  count = var.enable_aws ? 1 : 0
  role       = aws_iam_role.eks_node_role[0].name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

resource "aws_iam_role_policy_attachment" "eks_node_AmazonEKS_CNI_Policy" {
  count = var.enable_aws ? 1 : 0
  role       = aws_iam_role.eks_node_role[0].name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
}

resource "aws_iam_role_policy_attachment" "eks_node_AmazonEBSCSIDriverPolicy" {
  count = var.enable_aws ? 1 : 0
  role       = aws_iam_role.eks_node_role[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
}

resource "aws_iam_policy" "get_secret_value" {
    count = var.enable_aws ? 1 : 0
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
                    aws_secretsmanager_secret_version.jfrog_oidc_integration_secret_version[0].arn
                ]
            }
        ]
    })
}

resource "aws_iam_policy" "get_user_pool" {
    count = var.enable_aws ? 1 : 0
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
                    "arn:aws:cognito-idp:${var.region}:${data.aws_caller_identity.current[0].account_id}:userpool/*"
                ]
            }
        ]
    })
}

resource "aws_iam_role_policy_attachment" "eks_node_JFrogAllowGetSecretValuePolicy" {
  count = var.enable_aws ? 1 : 0
  role       = aws_iam_role.eks_node_role[0].name
  policy_arn = aws_iam_policy.get_secret_value[0].arn
}

resource "aws_iam_role_policy_attachment" "eks_node_JFrogAllowGetUserPoolPolicy" {
  count = var.enable_aws ? 1 : 0
  role       = aws_iam_role.eks_node_role[0].name
  policy_arn = aws_iam_policy.get_user_pool[0].arn
}
