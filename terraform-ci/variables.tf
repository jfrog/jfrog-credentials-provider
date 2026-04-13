variable "region" {
  default = "eu-central-1"
}

variable "create_eks_cluster" {
  description = "Set to true to create a new EKS cluster, false to use an existing one."
  type        = bool
  default     = false
}

# Details of the EKS cluster already created
variable "self_managed_eks_cluster" {
  description = "Details of the existing EKS cluster (required if create_eks_cluster is false)."
  type = object({
    name     = string
  })
  default = null
}

check "self_managed_eks_cluster_conditional_requirement" {
  assert {
    condition     = var.create_eks_cluster || var.self_managed_eks_cluster != null
    error_message = "When 'create_eks_cluster' is false, the 'self_managed_eks_cluster' variable must be set. Please provide the 'name' and 'subnet_ids' for the existing cluster."
  }
}

# WARNING: CIDR "0.0.0.0/0" is full public access to the cluster, you should use a more restrictive CIDR
variable "cluster_public_access_cidrs" {
  # default = ["0.0.0.0/0"]
  default = [] # Changed to empty list for security
  type    = list(string)
  description = "List of CIDR blocks to allow access to the EKS cluster's public endpoint. Defaults to an empty list (no access). Provide your IP CIDR for management e.g. [\"YOUR_IP/32\"]."
}

check "cluster_public_access_cidrs_conditional_usage" {
  assert {
    condition     = !var.create_eks_cluster || length(var.cluster_public_access_cidrs) > 0
    error_message = "When 'create_eks_cluster' is true, 'cluster_public_access_cidrs' must be a non-empty list."
  }
}

variable "cluster_name" {
  description = "The name for the EKS cluster."
  type        = string
  default     = null
}


check "cluster_name_conditional_requirement" {
  assert {
    condition     = !var.create_eks_cluster || (var.create_eks_cluster && var.cluster_name != null)
    error_message = "When 'create_eks_cluster' is true, 'cluster_name' must be set to provide a name for the new EKS cluster."
  }
}

variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster."
  type        = string
  default     = "1.34"
}


# The full URL to the JFrog Credential Provider binary
variable "jfrog_credential_provider_binary_url" {
  description = "The full URL to the JFrog Credential Provider binary. Example: 'https://releases.jfrog.io/artifactory/run/jfrog-credentials-provider/0.1.0-beta.1/jfrog-credential-provider-aws-linux' or your custom Artifactory URL."
  type        = string
}

# The JFrog Artifactory URL (the one that will be the EKS container registry)
variable "artifactory_url" {
  description = "The JFrog Artifactory URL (e.g., mycompany.jfrog.io) that will be the EKS container registry."
  type        = string
}

variable "artifactory_user" {
  # default = "admin" # Default removed
  description = "The JFrog Artifactory username for OIDC mapping or other configurations if needed. This user will be mapped via OIDC or assumed role."
  type        = string
}

variable "artifactory_user_wi" {
  # default = "admin" # Default removed
  description = "The JFrog Artifactory username for OIDC mapping or other configurations if needed. This user will be mapped via OIDC or assumed role."
  type        = string
}

# Set the authentication method to use for the JFrog Artifactory
# Supported values are "cognito_oidc" or "assume_role"
variable "authentication_method" {
  default = "cognito_oidc"
}

variable "jfrog_oidc_provider_name" {
  default = "jfrog-aws-oidc-provider"
}

variable "jfrog_namespace" {
  default = "jfrog"
}

# Node Group configurations
variable "node_group_instance_types" {
  description = "Instance types for the OIDC-enabled EKS managed node group."
  type        = list(string)
  default     = ["t4g.small"]
}

variable "node_group_min_size" {
  description = "Minimum number of nodes for the OIDC node group."
  type        = number
  default     = 1
}

variable "node_group_max_size" {
  description = "Maximum number of nodes for the OIDC node group."
  type        = number
  default     = 1
}

variable "node_group_desired_size" {
  description = "Desired number of nodes for the OIDC node group."
  type        = number
  default     = 1
}

variable "daemonset_node_group_instance_types" {
  description = "Instance types for the DaemonSet test EKS managed node group."
  type        = list(string)
  default     = ["t4g.medium"]
}

variable "daemonset_node_group_min_size" {
  description = "Minimum number of nodes for the DaemonSet node group."
  type        = number
  default     = 1
}

variable "daemonset_node_group_max_size" {
  description = "Maximum number of nodes for the DaemonSet node group."
  type        = number
  default     = 2
}

variable "daemonset_node_group_desired_size" {
  description = "Desired number of nodes for the DaemonSet node group."
  type        = number
  default     = 1
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

variable "busybox_image_ds" {
  description = "Container image for the busybox pod on the daemonset node group for testing."
  type        = string
  default     = "partnership-docker-remote-test.jfrog.io/busybox:latest"
}

variable "ami_type" {
  description = "AMI type for the EKS node group."
  type        = string
  default     = "AL2023_ARM_64_STANDARD"
}

variable node_security_group_ids {
  description = "List of security group IDs for the EKS node groups. Required if create_eks_cluster is false."
  type        = list(string)
  default     = []
}

# Cloud Provider Enable Flags
variable "enable_aws" {
  description = "Enable AWS EKS infrastructure and testing"
  type        = bool
  default     = false
}

variable "enable_azure" {
  description = "Enable Azure AKS infrastructure and testing"
  type        = bool
  default     = false
}

# Azure Variables
variable "azure_subscription_id" {
  description = "Azure subscription ID."
  type        = string
  default     = null
}

variable "azure_resource_group_name" {
  description = "Name of the Azure resource group (must already exist)."
  type        = string
  default     = null
}

variable "azure_location" {
  description = "Azure region/location for resources."
  type        = string
  default     = "East US"
}

variable "existing_azure_app_client_id" {
  description = "Client ID of an existing Azure AD application to use. If not provided, a new application will be created."
  type        = string
  default     = null
}

variable "create_aks_cluster" {
  description = "Set to true to create a new AKS cluster, false to use an existing one."
  type        = bool
  default     = false
}

variable "aks_cluster_name" {
  description = "Name of the AKS cluster to create or use."
  type        = string
  default     = null
}

variable "azure_cluster_public_access_cidrs" {
  description = "List of CIDR blocks to allow access to the AKS cluster's public endpoint."
  type        = list(string)
  default     = []
}

variable "azure_node_count" {
  description = "Initial number of nodes for the AKS default node pool."
  type        = number
  default     = 1
}

variable "azure_node_vm_size" {
  description = "VM size for AKS nodes."
  type        = string
  default     = "Standard_D2s_v3"
}

variable "azure_admin_username" {
  description = "Admin username for AKS nodes."
  type        = string
  default     = "azureadmin"
}


check "azure_subscription_id_required" {
  assert {
    condition     = !var.enable_azure || (var.azure_subscription_id != null && var.azure_subscription_id != "")
    error_message = "When enable_azure is true, azure_subscription_id must be provided."
  }
}

check "azure_resource_group_required" {
  assert {
    condition     = !var.enable_azure || (var.azure_resource_group_name != null && var.azure_resource_group_name != "")
    error_message = "When enable_azure is true, azure_resource_group_name must be provided."
  }
}

check "aks_cluster_name_required" {
  assert {
    condition     = !var.enable_azure || !var.create_aks_cluster || (var.aks_cluster_name != null && var.aks_cluster_name != "")
    error_message = "When enable_azure is true and create_aks_cluster is true, aks_cluster_name must be provided."
  }
}

variable "artifactory_glob_pattern" {
  description = "The glob pattern for the Artifactory URL. This is used to match the images that the JFrog Credential Provider will be used for."
  type        = string
  default     = "partnership*.jfrog.io"
}