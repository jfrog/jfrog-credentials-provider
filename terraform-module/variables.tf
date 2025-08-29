# Azure only supports the DaemonSet installation method
variable "cloud_provider" {
  description = "Cloud provider to use for the JFrog Credential Provider."
  type        = string
  default     = "aws"
}

# AWS Only
variable "create_eks_node_groups" {
  description = "Flag to create EKS node groups."
  type        = bool
  default     = true
}

# AWS Only
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
      security_group_ids = optional(list(string), [])
      desired_size    = number
      max_size        = number
      min_size        = number
      ami_type        = string
      instance_types  = list(string)
      labels         = optional(map(string))
      taints         = optional(map(object({
        key    = string
        value  = string
        effect = string
      })), {}) 
    }))
  })

  default = null
}

# AWS Only
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

// define a free kubernetes object that can have any properties
variable "kubernetes_auth_object" {
  description = "A free-form Kubernetes object that can have any properties."
  type        =  object({
    host = optional(string, "")
    cluster_ca_certificate = optional(string, "")
    token = optional(string, "")
  })
  default     = {}
}

# Applicable for AWS and Azure
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
    default     = ""
}

# TODO
# Common variables for JFrog Credential Provider
variable "region" {
  default = "eu-central-1"
}

# This adds "-x86_64" or "-arm64" as suffix to the binary name. 
variable "jfrog_credential_provider_binary_url" {
  # Change default to git latest release
    description = "The full URL to the JFrog Credential Provider binary. Example: 'https://releases.jfrog.io/artifactory/run/jfrog-credentials-provider/0.1.0-beta.1/jfrog-credential-provider-aws-linux' or your custom Artifactory URL."
    type        = string
    default     = "https://releases.jfrog.io/artifactory/run/jfrog-credentials-provider/0.1.0-beta.1/jfrog-credential-provider-aws-linux"
}

variable "artifactory_url" {
    description = "The JFrog Artifactory URL (e.g., mycompany.jfrog.io) that will be the EKS container registry."
    type        = string
}

variable "artifactory_user" {
    description = "The JFrog Artifactory username for OIDC mapping or other configurations if needed. This user will be mapped via OIDC or assumed role."
    type        = string
}

# Applicable for AWS only
# Set the authentication method to use for the JFrog Artifactory in AWS
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

## Applicable for AWS and Azure
## Following variables are used for the OIDC authentication method
## It is expected that the AWS Cognito User Pool and Resource Server are already created
## and the necessary permissions are granted to the JFrog Credential Provider IAM Role (worker role)
## client ID and secret is expected to be part of the aws secret 
variable "jfrog_oidc_provider_name" {
    default = "jfrog-aws-oidc-provider"
}

# AWS ENVs for the JFrog Credential Provider
variable "aws_envs" {
  description = "Environment variables for the AWS Credential Provider."
  type = object({
    # AWS Cognito User Pool Secret Name
    aws_cognito_user_pool_secret_name = string
    # AWS Cognito User Pool Name
    aws_cognito_user_pool_name = string
    # AWS Cognito User Pool ID
    aws_cognito_user_pool_id = string
    # AWS Cognito User Pool Client ID
    aws_cognito_user_pool_client_id = string
    # AWS Cognito User Pool Domain Name
    aws_cognito_user_pool_domain_name = string
    # AWS Cognito Resource Server Name
    aws_cognito_resource_server_name = string
  })
  default = null
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

variable "aws_cognito_user_pool_id" {
    description = "The ID of the AWS Cognito User Pool"
    type        = string
    default     = null
}

variable "aws_cognito_user_pool_client_id" {
    description = "The client ID of the AWS Cognito User Pool"
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

variable "iam_role_arn" {
  description = "The ARN of the IAM role to be used by Jfrog Credential Provider"
  type        = string
  default     = null  
}

# Container Images
variable "alpine_tools_image" {
  description = "Container image for alpine-with-tools used in the injector init container."
  type        = string
  default     = "releases-docker.jfrog.io/jfrog/alpine-with-tools:3.21.0"
}

variable "pause_image" {
  description = "Container image for the pause container."
  type        = string
  default     = "releases-docker.jfrog.io/pause:3.7"

}

variable "wait_for_creation" {
  description = "Used as an alternative to depends-on since this is treated as a legacy module."
  type = string
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
      (var.aws_cognito_user_pool_id != null && var.aws_cognito_user_pool_id != "") &&
      (var.aws_cognito_user_pool_domain_name != null && var.aws_cognito_user_pool_domain_name != "") &&
      (var.aws_cognito_resource_server_name != null && var.aws_cognito_resource_server_name != "")
    )
    error_message = "If authentication_method is 'cognito_oidc', then 'jfrog_oidc_provider_name', 'aws_cognito_user_pool_secret_name', 'aws_cognito_user_pool_name', 'aws_cognito_user_pool_id', 'aws_cognito_user_pool_domain_name', 'aws_cognito_resource_server_name' must be provided and be non-empty strings."
  }
}

variable "azure_envs" {
  description = "Environment variables for the Azure Credential Provider."
  type = object({
    # Azure registered application client id
    azure_app_client_id = string
    # Azure Tenant ID
    azure_tenant_id = string
    # Azure App Audience
    azure_app_audience = string
    # Azure Nodepool Client ID
    azure_nodepool_client_id = string
  })
  default = null
}