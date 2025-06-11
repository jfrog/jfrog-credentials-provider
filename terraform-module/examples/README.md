# JFrog Credential Provider Examples

This directory contains example configurations for the three deployment methods supported by the JFrog Credential Provider Terraform module.

## Deployment Methods

The JFrog Credential Provider can be deployed using one of the following methods:

1. **EKS Node Group Creation** - Creates new EKS node groups with the JFrog Credential Provider pre-installed.
2. **DaemonSet Installation** - Deploys the JFrog Credential Provider to existing EKS clusters using a Kubernetes DaemonSet.
3. **AWS CLI Command Generation** - Generates AWS CLI commands to create Launch Templates with JFrog Credential Provider configuration.

## Example Structure

This directory contains two types of examples:

- **Configuration Examples**: Simple variable configuration files for different authentication methods
  - `terraform.tfvars.oidc` - Example configuration with Cognito OIDC authentication
  - `terraform.tfvars.assume_role` - Example configuration with AWS IAM Role authentication

- **Complete Module Example**: A full example implementation using the module
  - `module_example/` - Example using the module with Assume Role authentication and EKS Node Group creation

## How to Use

### Using the Example Configuration Files

1. Clone the repository containing the JFrog Credential Provider Terraform module.
2. Select either `terraform.tfvars.oidc` or `terraform.tfvars.assume_role` based on your authentication method.
3. Copy the selected file to `terraform.tfvars` in your own directory.
4. Customize the configuration for your environment, uncommenting the deployment method you want to use.
5. Run the following commands:

```bash
terraform init
terraform plan
terraform apply
```

### Using the Complete Module Example

A complete example with module reference is provided in the `module_example` directory:

```bash
cd module_example
# Edit terraform.tfvars to customize for your environment
terraform init
terraform plan
terraform apply
```

## Authentication Methods

The examples demonstrate both supported authentication methods:

1. **COGNITO OIDC** - Uses AWS Cognito for OIDC authentication with JFrog Artifactory.
2. **ASSUME ROLE** - Uses AWS IAM Role assumption for authentication with JFrog Artifactory.

Choose the authentication method that best suits your environment and requirements.

## Prerequisites

Before running these examples, ensure you have:

1. AWS credentials configured.
2. Terraform installed.
3. For the DaemonSet method: A running Kubernetes cluster and kubectl access.
4. For the OIDC method: A configured AWS Cognito User Pool.
5. For the Assume Role method: An IAM role with appropriate permissions.

## Example Variables Structure

All example configurations follow this structure:

```hcl
# Authentication Method (cognito_oidc or assume_role)
authentication_method = "assume_role" or "cognito_oidc"

# Deployment Method (uncomment ONE of these)
create_eks_node_groups = true
# jfrog_credential_plugin_daemonset_installation = true
# generate_aws_cli_command = true

# Authentication-specific variables...

# Deployment method-specific variables...
```

## Testing the Deployment

After deploying the JFrog Credential Provider, test it by launching a pod that pulls an image from your JFrog Artifactory:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-jfrog-credentials
spec:
  containers:
  - name: test-container
    image: example.jfrog.io/docker-local/my-image:latest
```

Apply this pod configuration:

```bash
kubectl apply -f test-pod.yaml
```

If the pod successfully pulls the image, the JFrog Credential Provider is working correctly.
