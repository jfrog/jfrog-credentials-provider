# JFrog Kubelet Credential Provider - CI/CD Testing Environment

This directory contains Terraform configuration for deploying a complete testing environment for the JFrog Kubelet Credential Provider. It can create new AWS EKS or Azure AKS clusters, or use existing ones, with the JFrog Kubelet Credential Provider pre-configured for continuous integration and testing purposes.

## Overview

This CI environment is designed for:
- Testing JFrog Kubelet Credential Provider functionality
- Validating provider deployments in CI/CD pipelines
- Providing a reproducible testing environment for development
- Demonstrating the integration between AWS EKS/Azure AKS and JFrog Artifactory

Note - This is not recommended for production.

## Deployment Options

**Note** - You need to export your Artifactory token if running locally. This is used to create IAM role mappings to the user and/or OIDC mappings to the provider.

```hcl
export ARTIFACTORY_TOKEN=<TOKEN>
```

The configuration supports deployment on both AWS and Azure with various scenarios:

### AWS Deployment Options

#### 1. Create New EKS Cluster (`enable_aws = true`, `create_eks_cluster = true`)
- Creates a complete EKS cluster with worker nodes
- Sets up VPC, subnets, and all networking components
- Configures IAM roles and security groups
- Installs JFrog Kubelet Credential Provider on all nodes

#### 2. Use Existing EKS Cluster (`enable_aws = true`, `create_eks_cluster = false`)
- Uses your existing EKS cluster
- Adds new worker nodes with JFrog Kubelet Credential Provider
- Creates only the necessary IAM roles and node groups
- Leaves existing infrastructure unchanged

### Azure Deployment Options

#### 1. Create New AKS Cluster (`enable_azure = true`, `create_aks_cluster = true`)
- Creates a complete AKS cluster with worker nodes
- Uses an existing Azure resource group
- Configures Azure AD applications and managed identities
- Installs JFrog Kubelet Credential Provider via DaemonSet

#### 2. Use Existing AKS Cluster (`enable_azure = true`, `create_aks_cluster = false`)
- Uses your existing AKS cluster
- Deploys JFrog Kubelet Credential Provider via DaemonSet
- Creates only the necessary Azure AD applications and identity mappings
- Leaves existing infrastructure unchanged

## Quick Start Examples

Choose the example that matches your needs:

### AWS Examples
- **[Create New EKS Cluster](./terraform.create_cluster.tfvars)** - Complete new EKS cluster setup
- **[Use Existing EKS Cluster](./terraform.existing_cluster.tfvars)** - Add to existing EKS cluster (set `create_eks_cluster = false`)

### Azure Examples  
- **[Create New AKS Cluster](./terraform.azure.tfvars)** - Complete new AKS cluster setup(set `create_aks_cluster = false`) for existing clusters


Each example includes:
- Pre-configured `terraform.tfvars` file
- Detailed setup instructions
- Prerequisites and usage notes

## Prerequisites

### Required Tools
- **Terraform**: Follow the [Install Terraform](https://developer.hashicorp.com/terraform/install) guide
- **kubectl**: For cluster management and testing

### For AWS Deployments
- **AWS CLI**: Follow the [AWS CLI Configuration](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html) guide

### For Azure Deployments  
- **Azure CLI**: Follow the [Azure CLI Installation](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) guide

### Cloud Provider Permissions

#### AWS Permissions
Ensure your AWS credentials have permissions for:
- **For New Clusters**: EKS, VPC, EC2, IAM (full cluster creation permissions)
- **For Existing Clusters**: EKS node group creation, IAM role creation

#### Azure Permissions
Ensure your Azure credentials have permissions for:
- **For New Clusters**: AKS, Resource Groups, Azure AD applications, managed identities
- **For Existing Clusters**: AKS management, Azure AD application creation

### JFrog Requirements
- Access to a JFrog Artifactory instance
- Admin permissions to configure IAM role mappings (AWS) or OIDC providers (Azure)
- Access token for API calls

## Configuration

### Core Configuration Variables

The main configuration is controlled by cloud provider selection and cluster creation flags:

**AWS Configuration:**
```hcl
# Enable AWS and set cluster creation option
enable_aws = true
create_eks_cluster = true  # or false

# Required when create_eks_cluster = true
cluster_name = "jfrog-test-cluster"
cluster_public_access_cidrs = ["YOUR_IP/32"]

# Required when create_eks_cluster = false  
self_managed_eks_cluster = {
  name = "your-existing-cluster-name"
}

# Common AWS configuration
artifactory_url = "example.jfrog.io"
artifactory_user = "aws-eks-user"
```

**Azure Configuration:**
```hcl
# Enable Azure and set cluster creation option
enable_azure = true
create_aks_cluster = true  # or false

# Azure subscription and resource group (must exist)
azure_subscription_id     = "your-azure-subscription-id"
azure_resource_group_name = "your-existing-resource-group"
azure_location           = "East US"

# Required when create_aks_cluster = true
aks_cluster_name = "jfrog-ci-aks-cluster"
azure_cluster_public_access_cidrs = ["YOUR_IP/32"]

# Required when create_aks_cluster = false
# aks_cluster_name should be set to existing cluster name

# Common Azure configuration
artifactory_url = "example.jfrog.io"
artifactory_user = "azure-aks-user"
```

### Setup Steps

1. **Choose your deployment method** and copy the appropriate example:
   ```bash
   # For AWS new cluster
   cp terraform.aws.tfvars.example terraform.tfvars
   
   # For Azure new cluster
   cp terraform.azure.tfvars.example terraform.tfvars
   
   # For both AWS and Azure
   cp terraform.both.tfvars.example terraform.tfvars
   ```

2. **Edit the configuration** file with your specific values:
   - Update `artifactory_url` with your JFrog instance
   - Set `artifactory_user` for role/identity mapping
   - Configure IP access restrictions
   - Set cluster names appropriately
   - For Azure: ensure resource group exists and set subscription ID

3. **Deploy the infrastructure**:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## Clean Up
To clean up the resources created by the Terraform code, run the following command
```shell
terraform destroy
```

## Examples Directory

The examples directory contains ready-to-use configurations for different deployment scenarios:

- **`terraform.aws.tfvars.example`**: Complete example for AWS EKS (create or existing cluster)
- **`terraform.azure.tfvars.example`**: Complete example for Azure AKS (create or existing cluster)  
- **`terraform.both.tfvars.example`**: Example for deploying to both AWS and Azure simultaneously

These examples provide a quick way to get started without manually configuring all the variables.

## Quick Setup

If you'd want to avoid reading all of this and just want to go ahead and try it then:

### For AWS:
1. Copy `terraform.aws.tfvars.example` to `terraform.tfvars`
2. Update the values based on your AWS Infrastructure
3. `terraform init` 
4. `terraform plan`

### For Azure:
1. Copy `terraform.azure.tfvars.example` to `terraform.tfvars`
2. Update the values based on your Azure Infrastructure  
3. `terraform init`
4. `terraform plan`

This creates testing environments with the JFrog Kubelet Credential Provider installed and configured. For AWS, it creates two node groups - one installs the plugin directly through user data, and the other uses DaemonSet installation. For Azure, it uses DaemonSet installation. Both launch test pods to verify functionality. 
