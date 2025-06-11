# JFrog Credential Provider CI Examples

This directory contains example configurations for deploying the JFrog Credential Provider in different scenarios.

## Available Examples

### [Create New Cluster](./create-cluster/)
- **Use Case**: Testing from scratch, CI/CD environments, demos
- **What it does**: Creates a complete new EKS cluster with JFrog Credential Provider
- **Resources Created**: VPC, subnets, EKS cluster, worker nodes, IAM roles
- **Time to Deploy**: ~15-20 minutes

### [Existing Cluster](./existing-cluster/)  
- **Use Case**: Adding to production clusters, existing infrastructure
- **What it does**: Adds JFrog Credential Provider to your existing EKS cluster
- **Resources Created**: New worker node group, IAM roles
- **Time to Deploy**: ~5-10 minutes

## Choosing the Right Example

| Scenario | Example to Use | Notes |
|----------|---------------|--------|
| First time testing | `create-cluster` | Creates everything you need |
| CI/CD pipeline | `create-cluster` | Clean environment each time |
| Production integration | `existing-cluster` | Doesn't affect existing workloads |
| Development/Testing | Either | Depends on your preference |

## Quick Start

1. **Choose an example** based on your needs
2. **Navigate to the example directory**:
   ```bash
   cd create-cluster  # or existing-cluster
   ```
3. **Follow the README** in that directory for detailed instructions

## Common Prerequisites

All examples require:
- AWS CLI configured with appropriate permissions
- Terraform installed (>= 1.0)
- JFrog Artifactory instance and access token
- Your IP address for cluster access (create-cluster only)

## Cost Considerations

- **Create Cluster**: Higher cost (full EKS cluster + VPC resources)
- **Existing Cluster**: Lower cost (additional node group only)

Remember to run `terraform destroy` when finished testing to avoid ongoing charges.
