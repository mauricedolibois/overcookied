# Route 53 Configuration for Custom Domain: overcookied.de
# 
# STEP 1: ACQUIRE THE DOMAIN
# ==========================
# Since your AWS account is Free Tier, you cannot register domains through Route 53.
# 
# Options to acquire overcookied.de:
# 
# A) Register through a German registrar (cheaper for .de domains):
#    - IONOS (1und1): https://www.ionos.de/domains/de-domain
#    - Strato: https://www.strato.de/domains/
#    - United Domains: https://www.united-domains.de/
#    - Namecheap: https://www.namecheap.com/
#    
# B) Register through Route 53 (requires non-Free-Tier account):
#    - AWS Console → Route 53 → Registered Domains → Register Domain
#    - .de domains cost ~$9/year through Route 53
#
# STEP 2: CREATE HOSTED ZONE IN AWS (after acquiring domain)
# ===========================================================
# After registering the domain elsewhere, create a hosted zone in AWS:
#
# aws route53 create-hosted-zone --name overcookied.de --caller-reference "overcookied-$(Get-Date -Format 'yyyyMMddHHmmss')"
#
# Then update the NS records at your registrar to point to the AWS nameservers.

# ============================================
# TERRAFORM CONFIGURATION (uncomment when ready)
# ============================================

variable "domain_name" {
  description = "The root domain name"
  type        = string
  default     = "overcookied.de"
}

variable "app_subdomain" {
  description = "The subdomain for the application"
  type        = string
  default     = "app"  # Results in app.overcookied.de
}

# Hosted Zone (create this first, then update NS records at registrar)
# resource "aws_route53_zone" "main" {
#   name = var.domain_name
#   
#   tags = {
#     Project     = "overcookied"
#     Environment = "production"
#   }
# }

# ACM Certificate for HTTPS (must be in the same region as ALB)
# resource "aws_acm_certificate" "main" {
#   domain_name               = var.domain_name
#   subject_alternative_names = ["*.${var.domain_name}"]
#   validation_method         = "DNS"
#   
#   lifecycle {
#     create_before_destroy = true
#   }
#   
#   tags = {
#     Project     = "overcookied"
#     Environment = "production"
#   }
# }

# DNS validation records for ACM
# resource "aws_route53_record" "acm_validation" {
#   for_each = {
#     for dvo in aws_acm_certificate.main.domain_validation_options : dvo.domain_name => {
#       name   = dvo.resource_record_name
#       record = dvo.resource_record_value
#       type   = dvo.resource_record_type
#     }
#   }
#   
#   allow_overwrite = true
#   name            = each.value.name
#   records         = [each.value.record]
#   ttl             = 60
#   type            = each.value.type
#   zone_id         = aws_route53_zone.main.zone_id
# }

# Certificate validation
# resource "aws_acm_certificate_validation" "main" {
#   certificate_arn         = aws_acm_certificate.main.arn
#   validation_record_fqdns = [for record in aws_route53_record.acm_validation : record.fqdn]
# }

# A record for app.overcookied.de → ALB
# This uses external-dns or manual configuration with the Kubernetes Ingress ALB
# 
# Option 1: Manual record (requires ALB DNS name)
# resource "aws_route53_record" "app" {
#   zone_id = aws_route53_zone.main.zone_id
#   name    = "${var.app_subdomain}.${var.domain_name}"
#   type    = "A"
#   
#   alias {
#     name                   = data.aws_lb.ingress.dns_name
#     zone_id                = data.aws_lb.ingress.zone_id
#     evaluate_target_health = true
#   }
# }
#
# Option 2: Use external-dns controller (recommended)
# This automatically creates DNS records from Kubernetes Ingress annotations
# See: https://github.com/kubernetes-sigs/external-dns

# Output the nameservers to configure at registrar
# output "nameservers" {
#   description = "Configure these nameservers at your domain registrar"
#   value       = aws_route53_zone.main.name_servers
# }

