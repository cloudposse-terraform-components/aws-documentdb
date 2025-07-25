resource "aws_ssm_parameter" "master_username" {
  count = local.enabled ? 1 : 0

  name  = "/${module.this.name}/master_username"
  type  = "String"
  value = var.master_username
}

resource "random_password" "master_password" {
  count = local.enabled ? 1 : 0

  # character length
  length = 33

  special = false
  upper   = true
  lower   = true
  number  = true

  min_special = 0
  min_upper   = 1
  min_lower   = 1
  min_numeric = 1
}

resource "aws_ssm_parameter" "master_password" {
  count = local.enabled ? 1 : 0

  name  = "/${module.this.name}/master_password"
  type  = "SecureString"
  value = one(random_password.master_password[*].result)
}
