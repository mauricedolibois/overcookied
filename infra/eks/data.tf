# Data source to read Base layer outputs
data "terraform_remote_state" "base" {
  backend = "s3"
  config = {
    bucket = "overcookied-terraform-state"
    key    = "base/terraform.tfstate"
    region = var.aws_region
  }
}

# Get AWS account ID
data "aws_caller_identity" "current" {}

# Locals for easier access to base layer outputs
locals {
  vpc_id             = data.terraform_remote_state.base.outputs.vpc_id
  public_subnet_ids  = data.terraform_remote_state.base.outputs.public_subnet_ids
  ecr_backend_url    = data.terraform_remote_state.base.outputs.ecr_backend_url
  ecr_frontend_url   = data.terraform_remote_state.base.outputs.ecr_frontend_url
  account_id         = data.aws_caller_identity.current.account_id
  cluster_name       = "${var.project_name}-eks"
}
