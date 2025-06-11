resource "local_file" "pre_bootstrap_user_data" {
    count = var.generate_aws_cli_command ? 1 : 0
    content  = <<-EOF
        Content-Type: multipart/mixed; boundary="//"
        MIME-Version: 1.0

        --//
        Content-Transfer-Encoding: 7bit
        Content-Type: text/x-shellscript
        Mime-Version: 1.0


        echo '${local.jfrog_provider_content}' > /etc/eks/image-credential-provider/jfrog-credential-config.json

        export JFROG_CREDENTIAL_PROVIDER_BINARY_URL="${var.jfrog_credential_provider_binary_url}"
        export ARTIFACTORY_URL="${var.artifactory_url}"
        ${file("${path.module}/jfrog/bootstrap.sh")}
        --//--
        EOF
    filename = "${path.module}/pre_bootstrap_user_data.sh"
}
resource "local_file" "launch_template_data_json_file" {
    count = var.generate_aws_cli_command ? 1 : 0
    content = jsonencode({
        UserData = base64encode(local_file.pre_bootstrap_user_data[0].content)
    }) 
    filename = "${path.module}/generated_launch_template_data.json"
}

output "create_launch_template_aws_cli_command" {
    description = "AWS CLI command to create an EC2 Launch Template using a generated JSON file for launch template data."
    value = var.generate_aws_cli_command ? (<<CMD
aws ec2 create-launch-template \
--launch-template-name "JfrogKubeletCredentialPluginNodeLT" \
--version-description "Initial version for kubelet plugin nodes" \
--region "${var.region}" \
--launch-template-data "file://${local_file.launch_template_data_json_file[0].filename}"
    CMD
    ) : "AWS CLI command generation is disabled because 'generate_aws_cli_command' is false."
    sensitive = false
    depends_on = [local_file.launch_template_data_json_file[0]]
}