variable "create_eks_node_groups" {
  description = "Flag to create EKS node groups."
  type        = bool
  default     = true
}

# A NG will be created for each entry in this list
variable "eks_node_group_configuration" {
  description = "Configuration for EKS node groups"
  type = object({
    node_role_arn                   = string
    cluster_name                    = string
    cluster_service_ipv4_cidr       = string
    subnet_ids                      = list(string)

    node_groups = list(object({
      name            = string
      desired_size    = number
      max_size        = number
      min_size        = number
      ami_type        = string
      instance_types  = list(string)
      labels         = optional(map(string))
      taints         = optional(list(object({
        key    = string
        value  = string
        effect = string
      })), [])
    }))
  })

  default = null
}

variable generate_aws_cli_command {
  description = "Flag to generate AWS CLI command for the Launch Template with the JFrog Kubelet Credential Plugin configuration."
  type        = bool
  default     = false
}

# variable aws_launch_template_configuration {
#     description = "Information about the AWS Launch Template to generate the CLI command for."
#     type = object({
#         image_id                = string
#         instance_type           = string
#     })
#     default = {
#         image_id         = "ami-019c4cd2e060955a6"
#         instance_type    = "t4g.medium"
#     }
# }


variable jfrog_credential_plugin_daemonset_installation {
  description = "Use DaemonSet installation to install for the JFrog Credential Plugin in Node groups"
  type        = bool
  default     = false
}

variable "daemonset_configuration" {
    description = "Configuration for the DaemonSet that installs the JFrog Credential Provider."
    type = object({
        jfrog_namespace = string
        node_selector = optional(list(map(string)))
        tolerations = optional(list(object({
            key      = string
            operator = string
            value    = string
            effect   = string
        })), [])
    })
    default = {
        jfrog_namespace = "jfrog"
        node_selector = []
        tolerations = []
    }
}

variable "kubeconfig_path" {
    description = "Path to the kubeconfig file for the EKS cluster."
    type        = string
    default     = "~/.kube/config"
}
# Common variables for JFrog Credential Provider
variable "region" {
  default = "eu-central-1"
}

# This adds "-x86_64" or "-arm64" as suffix to the binary name. 
variable "jfrog_credential_provider_binary_url" {
  # Change default to git latest release
    description = "The full URL to the JFrog Credential Provider binary. Example: 'https://releases.jfrog.io/jfrog_credentials_provider/jfrog-credential-provider-aws-linux-arm64' or your custom Artifactory URL."
    type        = string
    default     = "https://github.com/jfrog/jfrog-credentials-provider/releases/download/__JFROG_CREDENTIAL_PROVIDER_VERSION__/jfrog-credential-provider-aws-linux"
}

variable "artifactory_url" {
    description = "The JFrog Artifactory URL (e.g., mycompany.jfrog.io) that will be the EKS container registry."
    type        = string
}

variable "artifactory_user" {
    description = "The JFrog Artifactory username for OIDC mapping or other configurations if needed. This user will be mapped via OIDC or assumed role."
    type        = string
}

# Set the authentication method to use for the JFrog Artifactory
# Supported values are "cognito_oidc" or "assume_role"
variable "authentication_method" {
    description = "The authentication method to use for JFrog Artifactory. Supported values are 'cognito_oidc' or 'assume_role'."
    type        = string
    default     = "cognito_oidc"

    validation {
        condition     = contains(["cognito_oidc", "assume_role"], var.authentication_method)
        error_message = "Allowed values for authentication_method are 'cognito_oidc' or 'assume_role'."
    }
}

## Following variables are used for the OIDC authentication method
## It is expected that the AWS Cognito User Pool and Resource Server are already created
## and the necessary permissions are granted to the JFrog Credential Provider IAM Role (worker role)
## client ID and secret is expected to be part of the aws secret 
variable "jfrog_oidc_provider_name" {
    default = "jfrog-aws-oidc-provider"
}

# Must contain client ID and secret for aws cognito user pool
# {"client-secret":"__CLIENT_SECRET__",}",
# "client-id":"__CLIENT_ID__"}
variable "aws_cognito_user_pool_secret_name" {
  description = "The secret name for the AWS Cognito User Pool. This is used to authenticate the JFrog Credential Provider with AWS Cognito."
  type        = string
  default     = null
}

variable "aws_cognito_user_pool_name" {
    description = "The name of the AWS Cognito User Pool"
    type        = string
    default     = null
}

variable "aws_cognito_user_pool_domain_name" {
    description = "The domain name for the AWS Cognito User Pool"
    type        = string
    default     = null
}

variable "aws_cognito_resource_server_name" {
    description = "The name of the AWS Cognito Resource Server"
    type        = string
    default     = null
}

variable "artifactory_oidc_identity_mapping_username" {
  description = "The username in Artifactory to map the OIDC identity to. This user must exist in Artifactory."
  type        = string
  default     = null
}

variable "iam_role_arn" {
  description = "The ARN of the IAM role to be used by Jfrog Credential Provider"
  type        = string
  default     = null  
}

# Container Images
variable "alpine_tools_image" {
  description = "Container image for alpine-with-tools used in the injector init container."
  type        = string
  # Needs to be pushed to releases.jfrog.io
  default     = "eldada.jfrog.io/docker/alpine-with-tools:3.21.0"
}

variable "pause_image" {
  description = "Container image for the pause container."
  type        = string
  # Needs to be pushed to releases.jfrog.io
  default     = "gke.gcr.io/pause:3.7"
}

check "assume_role_requires_iam_role_arn" {
  assert {
    condition     = var.authentication_method != "assume_role" || (var.iam_role_arn != null && var.iam_role_arn != "")
    error_message = "If authentication_method is 'assume_role', then 'iam_role_arn' must be provided and be a non-empty string."
  }
}

check "cognito_oidc_requires_cognito_variables" {
  assert {
    condition = var.authentication_method != "cognito_oidc" || (
      (var.jfrog_oidc_provider_name != null && var.jfrog_oidc_provider_name != "") &&
      (var.aws_cognito_user_pool_secret_name != null && var.aws_cognito_user_pool_secret_name != "") &&
      (var.aws_cognito_user_pool_name != null && var.aws_cognito_user_pool_name != "") &&
      (var.aws_cognito_user_pool_domain_name != null && var.aws_cognito_user_pool_domain_name != "") &&
      (var.aws_cognito_resource_server_name != null && var.aws_cognito_resource_server_name != "") &&
      (var.artifactory_oidc_identity_mapping_username != null && var.artifactory_oidc_identity_mapping_username != "")
    )
    error_message = "If authentication_method is 'cognito_oidc', then 'jfrog_oidc_provider_name', 'aws_cognito_user_pool_secret_name', 'aws_cognito_user_pool_name', 'aws_cognito_user_pool_domain_name', 'aws_cognito_resource_server_name', and 'artifactory_oidc_identity_mapping_username' must be provided and be non-empty strings."
  }
}