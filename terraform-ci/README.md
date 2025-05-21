# JFrog Credential Provider on AWS POC
This directory has the code the Terraform configuration to deploy an AWS EKS cluster with the JFrog Credential Provider added to the nodes.

## Prerequisites
### Terraform
Follow the [Install Terraform](https://developer.hashicorp.com/terraform/install) page to install Terraform on your machine.

### AWS CLI
You need to have the AWS CLI installed and configured with your credentials.<br>
Follow the [AWS CLI Configuration](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html) guide to set up the AWS CLI.<br>
Configure the AWS CLI to your desired account and region.

## Setup
For deploying the EKS with the JFrog Credential Provider, you will need to set up the following

1. Create a [terraform.tfvars](terraform.tfvars) file with the following variables
```hcl
# AWS Region to be used
region = "eu-central-1"

# Allow access from JFrog IP addresses
cluster_public_access_cidrs = ["52.9.243.19/32","52.215.237.185/32","34.233.113.191/32","13.127.185.21/32"]

# The JFrog Credential Provider binary URL (no authentication required)
jfrog_credential_provider_binary_url = "https://eldada.jfrog.io/artifactory/public-local/jfrog_credentials_provider/jfrog-credential-provider-linux-arm64"

# The JFrog Artifactory URL (the one that will be the EKS container registry)
artifactory_url  = "eldada.jfrog.io"

# The JFrog Artifactory username that will be granted the assume role permission
artifactory_user = "aws-eks-user"

```

2. Prepare the terraform workspace
```shell
terraform init
```

3. Plan the deployment
```shell
terraform plan
```

4. Apply the deployment
```shell
terraform apply
```

5. After the deployment is complete, you will see the output with the EKS cluster details and the command to configure Artifactory with the AWS IAM role to use
```shell
# The command to configure Artifactory with the AWS IAM role
curl -XPUT -H "Authorization: Bearer <ACCESS_TOKEN>" \
    -H "Content-type: application/json" \
    https://eldada.jfrog.io/access/api/v1/aws/iam_role \
    -d '{"username":"aws-eks-user","iam_role":"arn:aws:iam::471112533590:role/eks-node-role-demo-eks-cluster-eu-west-1"}'
```

6. Test the JFrog Credential Provider integration by deploying a pod with a container from Artifactory
```shell
kubectl apply -f ../example/pod_example.yaml
```
If all is set up correctly, the pod will be deployed successfully.

## Clean Up
To clean up the resources created by the Terraform code, run the following command
```shell
terraform destroy
```
