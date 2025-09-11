⚠️ Beta Release Notice

This project is currently in its beta phase, meaning it's still under active development. We strongly recommend thorough testing in non-production environments before deployment to any production system.

# JFrog Kubelet Credential Provider

A [Kubernetes kubelet credential provider](https://kubernetes.io/docs/tasks/administer-cluster/kubelet-credential-provider/) **for Amazon EKS and Azure AKS** that enables seamless authentication with JFrog Artifactory for container image pulls, eliminating the need for manual image pull secret management.

> **Coming Soon**: Google Cloud GKE support is currently in development.

## Overview

The JFrog Kubelet Credential Provider leverages the native Kubernetes kubelet Credential Provider feature to dynamically retrieve credentials for pulling container images from JFrog Artifactory. This approach provides several key benefits:

- **No Image Pull Secrets**: Eliminates the need to create and manage Kubernetes secrets
- **Enhanced Security**: Credentials are retrieved dynamically rather than stored in etcd
- **Simplified Operations**: Reduces operational overhead for credential rotation and management
- **Native Integration**: Uses built-in Kubernetes capabilities for credential management

## How It Works

1. A pod is created with an image stored in JFrog Artifactory
2. Kubelet identifies the image URL matches the configured pattern for the JFrog Kubelet Credential Provider
3. Kubelet invokes the JFrog Kubelet Credential Provider binary
4. The provider authenticates with the cloud provider (AWS IAM roles/OIDC or Azure managed identities) and exchanges credentials with Artifactory
5. Valid registry credentials are returned to kubelet for the image pull

## Quick Start

The easiest way to deploy the JFrog Kubelet Credential Provider is using our Terraform module:

```bash
cd terraform-module
# Copy and customize one of the example configurations
cp examples/terraform.assume_role.tfvars terraform.tfvars
# Edit terraform.tfvars for your environment
terraform init
terraform plan
terraform apply
```

## Deployment Options

The JFrog Kubelet Credential Provider supports three deployment methods:

1. **EKS Node Groups** - Creates new node groups with the provider pre-installed
2. **DaemonSet** - Installs the provider on existing EKS clusters
3. **Launch Template Generation** - Generates AWS CLI commands for custom deployments

See the [terraform-module](./terraform-module) directory for detailed deployment instructions and examples.

## Authentication Methods

### AWS Authentication
- **AWS IAM Role Assumption**: Uses EC2 instance [IAM roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html) for authentication
- **AWS Cognito OIDC**: Uses OIDC tokens from [AWS Cognito](https://docs.aws.amazon.com/cognito/latest/developerguide/what-is-amazon-cognito.html) for authentication

**Note**: For AWS, You must select either IAM Role Assumption OR Cognito OIDC as your authentication method. 
They cannot be used simultaneously in the same deployment.

### Azure Authentication
- **Azure Managed Identity OIDC**: Uses [Azure managed identities](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview) with OIDC for authentication


## Requirements

- Amazon EKS cluster or Azure AKS cluster
- JFrog Artifactory instance
- Based on your chosen cloud provider and authentication method:
  - **For AWS IAM Role Assumption**: IAM role mapped to a JFrog Artifactory user
  - **For AWS Cognito OIDC**: OIDC provider and identity mappings
  - **For Azure Managed Identity**: Azure AD application for OIDC and kubelet Identity. 
  For more information, see [terraform-module](./terraform-module)


## Logging and Debugging

Plugin logs are available in your kubelet VM at:
```bash
tail -f /var/log/jfrog-credential-provider.log
```

For detailed debugging instructions, see the [debug doc](./debug.md) file.

