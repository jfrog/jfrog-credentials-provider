# JFrog Credential Provider - CI/CD Testing Environment

This directory contains Terraform configuration for deploying a complete testing environment for the JFrog Credential Provider. It can either create a new AWS EKS cluster or use an existing one, with the JFrog Credential Provider pre-configured for continuous integration and testing purposes.

## Overview

This CI environment is designed for:
- Testing JFrog Credential Provider functionality
- Validating provider deployments in CI/CD pipelines
- Providing a reproducible testing environment for development
- Demonstrating the integration between AWS EKS and JFrog Artifactory

Note - This is not recommended for production.

## Deployment Options

**Note** - You need to export your Artifactory token if running locally. This is used to create IAM role mappings to the user and/or OIDC mappings to the provider.

```hcl
export ARTIFACTORY_TOKEN=<TOKEN>
```


The configuration supports two deployment scenarios:

### 1. Create New EKS Cluster (`create_eks_cluster = true`)
- Creates a complete EKS cluster with worker nodes
- Sets up VPC, subnets, and all networking components
- Configures IAM roles and security groups
- Installs JFrog Credential Provider on all nodes

### 2. Use Existing EKS Cluster (`create_eks_cluster = false`)
- Uses your existing EKS cluster
- Adds new worker nodes with JFrog Credential Provider
- Creates only the necessary IAM roles and node groups
- Leaves existing infrastructure unchanged

## Quick Start Examples

Choose the example that matches your needs:

- **[Create New Cluster](./examples/create-cluster/)** - Complete new EKS cluster setup
- **[Use Existing Cluster](./examples/existing-cluster/)** - Add to existing EKS cluster

Each example includes:
- Pre-configured `terraform.tfvars` file
- Detailed setup instructions
- Prerequisites and usage notes

## Prerequisites

### Required Tools
- **Terraform**: Follow the [Install Terraform](https://developer.hashicorp.com/terraform/install) guide
- **AWS CLI**: Follow the [AWS CLI Configuration](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html) guide
- **kubectl**: For cluster management and testing

### AWS Permissions
Ensure your AWS credentials have permissions for:
- **For New Clusters**: EKS, VPC, EC2, IAM (full cluster creation permissions)
- **For Existing Clusters**: EKS node group creation, IAM role creation

### JFrog Requirements
- Access to a JFrog Artifactory instance
- Admin permissions to configure IAM role mappings
- Access token for API calls

## Configuration

### Core Configuration Variables

The main configuration is controlled by the `create_eks_cluster` variable:

```hcl
# Set to true to create a new EKS cluster, false to use existing
create_eks_cluster = true  # or false

# Required when create_eks_cluster = true
cluster_name = "jfrog-test-cluster"
cluster_public_access_cidrs = ["YOUR_IP/32"]

# Required when create_eks_cluster = false  
self_managed_eks_cluster = {
  name = "your-existing-cluster-name"
}

# Common configuration for both scenarios
artifactory_url = "example.jfrog.io"
artifactory_user = "aws-eks-user"
```

### Setup Steps

1. **Choose your deployment method** and copy the appropriate example:
   ```bash
   # For new cluster
   cp examples/erraform.tfvars.create_cluster ./terraform.tfvars
   
   # For existing cluster  
   cp examples/erraform.tfvars.existing_cluster ./terraform.tfvars
   ```

2. **Edit the configuration** file with your specific values:
   - Update `artifactory_url` with your JFrog instance
   - Set `artifactory_user` for role mapping
   - Configure IP access (for new clusters)
   - Set cluster name appropriately

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

The `examples/` directory contains ready-to-use configurations for both deployment scenarios:

- **`terraform.create_cluster.tfvars`**: Complete example for creating a new EKS cluster
- **`terraform.existing_cluster.tfvars`**: Example for using an existing EKS cluster

These examples provide a quick way to get started without manually configuring all the variables.

## Quick Setup

If you'd want to avoid reading all of this and just want to go ahead and try it then:
1. Copy `build/terraform.tfvars` to `terraform-ci`
2. `terraform init` 
3. `terraform plan`

This creates two node groups in an existing cluster. One installs Jfrog Credential Provider Plugin directly through user data.
The other node group is used for daemonset install. This will also launch two pods to verify if this works. 
