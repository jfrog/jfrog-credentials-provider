# Contributing to JFrog Kubelet Credential Provider

Thank you for your interest in contributing to the JFrog Kubelet Credential Provider! We welcome contributions from the community.

## How to Contribute

1. **Create a GitHub issue** for bugs or feature requests
2. **Comment on the issue** to indicate your intent to work on the feature/bug
3. **Create a branch** and make your changes
4. **Test your changes** using `terraform-ci` or `terraform-module` (see respective README files for guidance)
5. **Submit a Pull Request** with testing results

## Pull Request Requirements

Please include the following in your Pull Request:

- **Description**: Clear explanation of changes made and rationale
- **Testing Results**: Output from `terraform plan` and/or `terraform apply`
- **Configuration**: The `terraform.tfvars` file used for testing
- **Environment**: AWS region, Kubernetes version, and other relevant details

## Testing Guidelines

Before submitting your PR, ensure you have:
- Tested your changes locally using Terraform
- Verified the configuration works in your environment
- Included any error messages or issues encountered during testing

We appreciate your contributions and will review your PR as soon as possible!