# JFrog Credential Provider Module Example

This directory contains a complete implementation example of the JFrog Credential Provider Terraform module using the EKS Node Group deployment method with IAM Role authentication.

## Implementation Details

This example demonstrates:
- Setting up the JFrog Credential Provider with EKS Node Group deployment
- Using AWS IAM Role authentication method
- Creating a complete Terraform module structure

## Configuration Files

- `main.tf` - Main Terraform configuration with module reference
- `variables.tf` - Variable declarations
- `terraform.tfvars` - Example variable values (customize for your environment)

## How to Use

1. Review and customize the `terraform.tfvars` file with your own values
2. Initialize Terraform: `terraform init`
3. Review the planned changes: `terraform plan`
4. Apply the configuration: `terraform apply`

