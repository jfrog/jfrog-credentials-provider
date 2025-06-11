# Existing EKS Cluster Example

This example demonstrates how to add the JFrog Credential Provider to an existing EKS cluster.

## What This Example Does

- Uses your existing EKS cluster
- Creates additional worker nodes with JFrog Credential Provider
- Sets up IAM roles and policies for the new nodes
- Configures both assume-role and OIDC authentication methods

## Prerequisites

- AWS CLI configured with appropriate permissions
- Terraform installed
- An existing EKS cluster
- kubectl access to your existing cluster

## Usage

1. Copy and customize the configuration:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your specific values
   ```

2. Update the following required values in `terraform.tfvars`:
   - `self_managed_eks_cluster.name`: Your existing cluster name
   - `region`: The AWS region where your cluster is located
   - `artifactory_url`: Your JFrog Artifactory URL
   - `artifactory_user`: Your JFrog user for mapping

3. Deploy the additional infrastructure:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

4. Verify kubectl is configured for your cluster:
   ```bash
   kubectl get nodes
   ```

5. Map the IAM role to your JFrog Artifactory user (see output for exact command).

## Important Notes

- This example only adds new worker nodes to your existing cluster
- Your existing cluster and nodes remain unchanged
- The new nodes will have the JFrog Credential Provider pre-configured
- Make sure you have the necessary permissions to create node groups in your existing cluster
