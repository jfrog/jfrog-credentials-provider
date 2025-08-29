data "aws_availability_zones" "available" {
    count = var.cloud_provider == "aws" ? 1 : 0
    filter {
        name   = "opt-in-status"
        values = ["opt-in-not-required"]
    }
}