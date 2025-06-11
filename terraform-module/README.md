# JFrog Credential Provider Terraform Module

This Terraform module sets up the JFrog Credential Provider for Kubernetes clusters running on AWS. It supports two authentication methods (`cognito_oidc` and `assume_role`) and offers three different deployment methods to suit various infrastructure requirements.

## Overview

JFrog Credential Provider is a kubelet credential provider that enables Kubernetes nodes to securely authenticate with JFrog Artifactory for pulling container images. This module facilitates the deployment and configuration of the JFrog Credential Provider across your Kubernetes infrastructure.

## Features
- Configures the JFrog Credential Provider for Kubernetes clusters on AWS
- Supports `cognito_oidc` and `assume_role` authentication methods
- Offers three flexible deployment methods:
  1. EKS Node Group creation with built-in JFrog Credential Provider
  2. DaemonSet installation for existing EKS clusters
  3. AWS CLI command generation for custom instance provisioning

## Deployment Methods

Note - If you'd like to get right into running commands look at [Lazy Setup](#lazy-setup). But we recommend reading through all the possible configurations

### Required Configuration
Irrespective of the deployment method you decide, these variables are required by default. 

```hcl
artifactory_url = "myart.jfrog.io"
artifactory_user = "aws-eks-user"
// This should be mapped to artifactory user or the OIDC provider
iam_role_arn = "<ARN>"
```


### 1. EKS Node Group Creation
This method creates new EKS node groups with the JFrog Credential Provider pre-installed as part of the node bootstrap process. Use this method when provisioning new EKS node groups.

**Key Configuration Parameters:**
```hcl
create_eks_node_groups = true
// Node role arn can be same as iam_role_arn defined above.
// They can different too, but node role should be able to assume the iam_role_arn defined. 
eks_node_group_configuration = {
  node_role_arn             = "arn:aws:iam::123456789012:role/eks-node-role"
  cluster_name              = "my-eks-cluster"
  cluster_service_ipv4_cidr = "172.20.0.0/16"
  subnet_ids                = ["subnet-abc123", "subnet-def456"]
  
  node_groups = [{
    name           = "jfrog-enabled-group"
    desired_size   = 2
    max_size       = 4
    min_size       = 2
    ami_type       = "AL2_x86_64"
    instance_types = ["t3.medium"]
    labels         = { "jfrog-credential-provider" = "enabled" }
  }]
}
```

**How It Works:**
- Generates appropriate configuration files based on your authentication method
- Adds the JFrog Credential Provider configuration to the EKS node's bootstrap process
- Installs and configures the credential provider as part of node initialization
- Restarts the kubelet to enable the credential provider

### 2. DaemonSet Installation
This method deploys the JFrog Credential Provider to existing EKS clusters using a Kubernetes DaemonSet. Use this when you need to add the credential provider to already running clusters.

**Key Configuration Parameters:**
```hcl
create_eks_node_groups = false
jfrog_credential_plugin_daemonset_installation = true
daemonset_configuration = {
  jfrog_namespace = "jfrog"
  node_selector = [
    {
      key = "kubernetes.io/os"
      value = "linux"
    }
  ]
  tolerations = [
    {
      key      = "dedicated"
      operator = "Equal"
      value    = "jfrog"
      effect   = "NoSchedule"
    }
  ]
}
kubeconfig_path = "~/.kube/config"
```

**How It Works:**
- Creates a namespace for the JFrog Credential Provider resources
- Deploys ConfigMaps containing provider configuration and bootstrap scripts
- Launches a DaemonSet with privileged init containers to:
  - Download the JFrog Credential Provider binary
  - Configure the kubelet to use the credential provider
  - Restart the kubelet service
- Uses a lightweight pause container to maintain the DaemonSet lifecycle

### 3. AWS CLI Command Generation
This method generates AWS CLI commands to create Launch Templates that include the JFrog Credential Provider configuration. Use this for custom instance provisioning workflows or when using AWS AutoScaling Groups with Launch Templates.

**Key Configuration Parameters:**
```hcl
create_eks_node_groups = false
jfrog_credential_plugin_daemonset_installation = false
generate_aws_cli_command = true
```

**How It Works:**
- Creates a bootstrap script (`pre_bootstrap_user_data.sh`) containing the credential provider configuration
- Generates a JSON file with Launch Template data
- Outputs an AWS CLI command that can be used to create a Launch Template
- The command includes the user data script that will:
  - Download the JFrog Credential Provider binary
  - Configure the kubelet credential provider
  - Restart the kubelet service

## Authentication Methods

### COGNITO OIDC Authentication
Uses AWS Cognito for OIDC authentication with JFrog Artifactory.

**Required Variables for OIDC:**
```hcl
authentication_method = "cognito_oidc"
jfrog_oidc_provider_name = "jfrog-aws-oidc"
aws_cognito_user_pool_secret_name = "my-cognito-secret"
aws_cognito_user_pool_name = "my-cognito-pool"
aws_cognito_user_pool_id = "region_random"
aws_cognito_user_pool_client_id = "random11"
aws_cognito_user_pool_domain_name = "my-domain"
aws_cognito_resource_server_name = "my-resource-server"
artifactory_user = "aws-eks-user"
```

### ASSUME ROLE Authentication
Uses AWS IAM Role assumption for authentication with JFrog Artifactory.

**Required Variables for Assume Role:**
```hcl
authentication_method = "assume_role"
iam_role_arn = "arn:aws:iam::123456789012:role/jfrog-role"
artifactory_user = "aws-eks-user"
```

## Usage

### Basic Usage
1. Clone this repository
2. Create a `terraform.tfvars` file with your desired configuration
3. Run the following commands:

```bash
terraform init
terraform plan
terraform apply
```

### Module Usage
To use this module in your Terraform configuration, add the following:

```hcl
module "jfrog_credential_provider" {
  source = "path/to/this/module"

  # Choose a deployment method
  create_eks_node_groups = true
  # or
  jfrog_credential_plugin_daemonset_installation = true
  # or
  generate_aws_cli_command = true

  # Set authentication method and required parameters
  authentication_method = "assume_role"
  artifactory_url = "example.jfrog.io"
  iam_role_arn = "arn:aws:iam::123456789012:role/jfrog-role"
  
  # Add other method-specific variables as needed
}
```

## Example Configurations

The module includes several examples to help you get started:

- **Configuration Examples**:
  - `examples/terraform.tfvars.oidc` - Example with Cognito OIDC authentication
  - `examples/terraform.tfvars.assume_role` - Example with AWS IAM Role authentication

- **Complete Module Example**:
  - `examples/module_example/` - A complete implementation using the module with Assume Role authentication and EKS Node Group creation

To use these examples:
1. Clone the repository
2. Copy one of the example files to your working directory
3. Customize it for your environment
4. Run `terraform init`, `terraform plan`, and `terraform apply`

## Prerequisites

Before using this module, ensure you have:

1. **For EKS Node Group Method**:
   - AWS credentials with permissions to create and manage EKS node groups
   - An existing EKS cluster

2. **For DaemonSet Method**:
   - A running Kubernetes cluster
   - `kubectl` access to the cluster (via kubeconfig)

3. **For AWS CLI Command Method**:
   - AWS credentials with permissions to create Launch Templates
   - AWS CLI installed if you plan to execute the generated commands

4. **Authentication Requirements**:
   - For OIDC: A configured AWS Cognito User Pool with an appropriate resource server
   - For Assume Role: An IAM role with appropriate permissions and trust relationships

## Lazy Setup

If you'd want to avoid reading all of this and just want to go ahead and try it then:

1. Copy `build/terraform.tfvars.lazy` to `terraform-ci`
  ```
    cp terraform-module/examples/terraform.tfvars.lazy terraform-module/terraform.tfvars
  ```
2. Update the values based on your Infrastructure
2. `terraform init` 
3. `terraform apply`

## Important Notes

- Ensure the IAM role used has the necessary permissions (at minimum `sts:GetCallerIdentity`)
- For JFrog Artifactory integration, ensure the IAM role is mapped to a JFrog Artifactory user:
  ```
  curl -XGET -H "Content-type: application/json" -H "Authorization: Bearer <TOKEN>" \
    https://example.jfrog.io/access/api/v1/aws/iam_role
  
  ```
  If it isn't, Run - 
  ```
  curl -XPUT -H "Content-type: application/json" -H "Authorization: Bearer <TOKEN>" \
    https://example.jfrog.io/access/api/v1/aws/iam_role \
    -d '{"username":"artifactory-user", "iam_role": "arn:aws:iam::123456789012:role/eks-node-role"}'
  ```

- Similar integration for OIDC in artifactory
  ```
  curl -XGET -H "Authorization: Bearer <Token>" \ 
  https://<ARTIFACTORY_URL>/access/api/v1/oidc/<PROVIDER_NAME>

  curl -XGET -H "Authorization: Bearer <Token>" \ 
  https://<ARTIFACTORY_URL>/access/api/v1/oidc/<PROVIDER_NAME>/identity_mappings

  ```
  If these return empty. Run the following commands - 
  ```
  curl  -XPOST "https://<ARTIFACTORY_URL>/access/api/v1/oidc" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer <TOKEN>" \
      -d '{
        "name": "<PROVIDER_NAME>",
        "issuer_url": "https://cognito-idp.<REGION>.amazonaws.com/<USER_POOL_ID>/",
        "description": "OIDC with AWS ",
        "provider_type": "Generic OpenID Connect",
        "token_issuer": "https://cognito-idp.<REGION>.amazonaws.com/<USER_POOL_ID>",
        "use_default_proxy": false
      }'

  curl -XPOST "https://<ARTIFACTORY_URL>/access/api/v1/oidc/<PROVIDER_NAME>/identity_mappings" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer <TOKEN>" \
      -d '{
        "name": "<PROVIDER_NAME>",
        "description": "OIDC identity mapping",
        "claims": {
          "client_id": "<USER_POOL_CLIENT_ID>",
          "iss": "https://cognito-idp.<REGION>.amazonaws.com/<USER_POOL_ID>"
        },
        "token_spec": {
          "username": "<ARTIFACTORY_USER>",
          "scope": "applied-permissions/user",
          "audience": "*@*",
          "expires_in": 3600
        },
        "priority": 1
      }'
  ```
- After deployment, verify the provider is working by pulling an image from your JFrog Artifactory repository
