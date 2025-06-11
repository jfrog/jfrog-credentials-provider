# Create New EKS Cluster Example

This example demonstrates how to create a completely new EKS cluster with the JFrog Credential Provider pre-configured.

## What This Example Does

- Creates a new EKS cluster from scratch
- Sets up VPC, subnets, and networking
- Creates IAM roles and policies
- Deploys worker nodes with JFrog Credential Provider
- Configures both assume-role and OIDC authentication methods

## Prerequisites

- AWS CLI configured with appropriate permissions
- Terraform installed
- Your IP address for cluster access (update `cluster_public_access_cidrs`)

## Usage

1. Copy and customize the configuration:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your specific values
   ```

2. Update the following required values in `terraform.tfvars`:
   - `artifactory_url`: Your JFrog Artifactory URL
   - `artifactory_user`: Your JFrog user for mapping
   - `cluster_public_access_cidrs`: Your IP addresses for cluster access

3. Deploy the infrastructure:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

4. Configure kubectl:
   ```bash
   aws eks update-kubeconfig --region <your-region> --name jfrog-test-cluster
   ```

5. Map the IAM role to your JFrog Artifactory user (see output for exact command).

## Important Notes

- This example creates a complete AWS infrastructure, which incurs costs
- Make sure to update `cluster_public_access_cidrs` with your actual IP addresses
- The cluster will be publicly accessible from the specified CIDRs
- Remember to run `terraform destroy` when you're done testing
