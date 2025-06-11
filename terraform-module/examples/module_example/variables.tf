# Variables
variable "region" {
  description = "AWS region"
  default     = "us-west-2"
}

variable "artifactory_url" {
  description = "JFrog Artifactory URL"
  type        = string
}

variable "artifactory_user" {
  description = "JFrog Artifactory user for IAM role mapping"
  type        = string
}

variable "iam_role_arn" {
  description = "IAM role ARN for assume role authentication"
  type        = string
}

variable "node_role_arn" {
  description = "IAM role ARN for EKS node groups"
  type        = string
}

variable "cluster_name" {
  description = "EKS cluster name"
  type        = string
}

variable "cluster_service_ipv4_cidr" {
  description = "EKS cluster service IPv4 CIDR"
  default     = "172.20.0.0/16"
  type        = string
}

variable "subnet_ids" {
  description = "Subnet IDs for EKS node groups"
  type        = list(string)
}

variable "node_groups" {
  description = "EKS node groups configuration"
  type = list(object({
    name           = string
    desired_size   = number
    max_size       = number
    min_size       = number
    ami_type       = string
    instance_types = list(string)
    labels         = optional(map(string))
    taints         = optional(list(object({
      key    = string
      value  = string
      effect = string
    })), [])
  }))
  default = [
    {
      name           = "jfrog-enabled-ng"
      desired_size   = 2
      max_size       = 4
      min_size       = 1
      ami_type       = "AL2_x86_64"
      instance_types = ["t3.medium"]
      labels         = {
        "jfrog-credential-provider" = "enabled"
      }
    }
  ]
}

variable "jfrog_credential_provider_binary_url" {
  description = "URL to the JFrog Credential Provider binary"
  default     = "https://github.com/jfrog/jfrog-credentials-provider/releases/download/v0.1.0/jfrog-credential-provider-aws-linux"
  type        = string
}
