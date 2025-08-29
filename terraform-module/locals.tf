locals {
    iam_role_name = try(split("/", var.iam_role_arn)[length(split("/", var.iam_role_arn)) - 1], null)
    jfrog_provider_content = var.cloud_provider == "azure" ? local_file.jfrog_provider_azure[0].content : var.cloud_provider == "aws" ? (
            var.authentication_method == "assume_role" ? local_file.jfrog_provider_assume_role[0].content : local_file.jfrog_provider_oidc[0].content) : ""
}