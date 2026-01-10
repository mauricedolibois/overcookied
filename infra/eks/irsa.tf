# IAM Role for Backend Pod (DynamoDB + Secrets Manager access)
resource "aws_iam_role" "backend_pod" {
  name = "${var.project_name}-backend-pod-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = local.oidc_provider_arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_provider_url}:sub" = "system:serviceaccount:overcookied:backend-sa"
          "${local.oidc_provider_url}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = {
    Name = "${var.project_name}-backend-pod-role"
  }
}

# IAM Policy for DynamoDB access (Least Privilege)
resource "aws_iam_policy" "backend_dynamodb" {
  name        = "${var.project_name}-backend-dynamodb-policy"
  description = "DynamoDB access for Overcookied backend pods"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "DynamoDBAccess"
      Effect = "Allow"
      Action = [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:Query",
        "dynamodb:Scan",
        "dynamodb:BatchGetItem"
      ]
      Resource = [
        "arn:aws:dynamodb:${var.aws_region}:${local.account_id}:table/${var.dynamodb_table_users}",
        "arn:aws:dynamodb:${var.aws_region}:${local.account_id}:table/${var.dynamodb_table_games}",
        "arn:aws:dynamodb:${var.aws_region}:${local.account_id}:table/${var.dynamodb_table_games}/index/*"
      ]
    }]
  })
}

# IAM Policy for Secrets Manager access (Google OAuth credentials)
resource "aws_iam_policy" "backend_secrets" {
  name        = "${var.project_name}-backend-secrets-policy"
  description = "Secrets Manager access for Overcookied backend pods"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "SecretsManagerRead"
      Effect = "Allow"
      Action = [
        "secretsmanager:GetSecretValue"
      ]
      Resource = "arn:aws:secretsmanager:${var.aws_region}:${local.account_id}:secret:overcookied/*"
    }]
  })
}

# Attach policies to backend pod role
resource "aws_iam_role_policy_attachment" "backend_dynamodb" {
  role       = aws_iam_role.backend_pod.name
  policy_arn = aws_iam_policy.backend_dynamodb.arn
}

resource "aws_iam_role_policy_attachment" "backend_secrets" {
  role       = aws_iam_role.backend_pod.name
  policy_arn = aws_iam_policy.backend_secrets.arn
}

# IAM Role for AWS Load Balancer Controller
resource "aws_iam_role" "aws_load_balancer_controller" {
  name = "${var.project_name}-aws-load-balancer-controller"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = local.oidc_provider_arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_provider_url}:sub" = "system:serviceaccount:kube-system:aws-load-balancer-controller"
          "${local.oidc_provider_url}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = {
    Name = "${var.project_name}-aws-load-balancer-controller"
  }
}

# IAM Policy for AWS Load Balancer Controller
resource "aws_iam_policy" "aws_load_balancer_controller" {
  name        = "${var.project_name}-aws-load-balancer-controller-policy"
  description = "IAM policy for AWS Load Balancer Controller"

  policy = file("${path.module}/policies/aws-load-balancer-controller-policy.json")
}

resource "aws_iam_role_policy_attachment" "aws_load_balancer_controller" {
  role       = aws_iam_role.aws_load_balancer_controller.name
  policy_arn = aws_iam_policy.aws_load_balancer_controller.arn
}
