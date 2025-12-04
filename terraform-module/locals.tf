locals {
    iam_role_name = try(split("/", var.iam_role_arn)[length(split("/", var.iam_role_arn)) - 1], null)
    jfrog_provider_content = var.enable_azure ? local_file.jfrog_provider_azure[0].content : var.enable_aws ? (
            var.authentication_method == "assume_role" ? (
                var.aws_service_token_exchange && length(local_file.jfrog_provider_web_identity) > 0 ? local_file.jfrog_provider_web_identity[0].content : (
                    length(local_file.jfrog_provider_assume_role) > 0 ? local_file.jfrog_provider_assume_role[0].content : ""
                )
            ) : (
                length(local_file.jfrog_provider_oidc) > 0 ? local_file.jfrog_provider_oidc[0].content : ""
            )
        ) : ""
}