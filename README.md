# JFrog Credential Provider

A Kubernetes kubelet credential provider that enables seamless authentication with JFrog Artifactory for container image pulls in Amazon EKS, eliminating the need for manual image pull secret management.

## Overview

The JFrog Credential Provider leverages the native Kubernetes Kubelet Credential Provider feature to dynamically retrieve credentials for pulling container images from JFrog Artifactory. This approach provides several key benefits:

- **No Image Pull Secrets**: Eliminates the need to create and manage Kubernetes secrets
- **Enhanced Security**: Credentials are retrieved dynamically rather than stored in etcd
- **Simplified Operations**: Reduces operational overhead for credential rotation and management
- **Native Integration**: Uses built-in Kubernetes capabilities for credential management

## How It Works

1. A pod is created with an image stored in JFrog Artifactory
2. Kubelet identifies the image URL matches the configured pattern for the JFrog credential provider
3. Kubelet invokes the JFrog credential provider binary
4. The provider authenticates with AWS (using IAM roles or OIDC) and exchanges credentials with Artifactory
5. Valid registry credentials are returned to kubelet for the image pull

## Quick Start

The easiest way to deploy the JFrog Credential Provider is using our Terraform module:

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

The JFrog Credential Provider supports three deployment methods:

1. **EKS Node Groups** - Creates new node groups with the provider pre-installed
2. **DaemonSet** - Installs the provider on existing EKS clusters
3. **Launch Template Generation** - Generates AWS CLI commands for custom deployments

See the [terraform-module](./terraform-module) directory for detailed deployment instructions and examples.

## Authentication Methods

- **AWS IAM Role Assumption**: Uses EC2 instance IAM roles for authentication
- **AWS Cognito OIDC**: Uses OIDC tokens from AWS Cognito for authentication

**Note**: You must select either IAM Role Assumption OR Cognito OIDC as your authentication method - they cannot be used simultaneously in the same deployment.

## Requirements

- Amazon EKS cluster
- JFrog Artifactory instance
- Based on your chosen authentication method:
  - **For IAM Role Assumption**: IAM role mapped to a JFrog Artifactory user
  - **For Cognito OIDC**: OIDC provider and identity mappings
.. more details can be found in [terraform-module](./terraform-module)


## Logging and Debugging

Plugin logs are available in your kubelet VM at:
```bash
tail -f /var/log/jfrog-credential-provider.log
```

For detailed debugging instructions, see the [ref.doc](./to_be_entered_later) file.

