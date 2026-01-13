# Custom Domain Setup Guide for Overcookied

This guide explains how to configure `overcookied.de` (or any custom domain) for the Overcookied application.

## ✅ Free Tier Compatibility

**Yes, you can use a domain from IONOS (or any registrar) with AWS Free Tier!**

The Free Tier restriction only applies to **domain registration** through Route 53:

| Service | Free Tier Compatible? | Cost |
|---------|----------------------|------|
| Domain Registration (Route 53) | ❌ No | N/A - not allowed |
| Domain Registration (IONOS) | ✅ Yes (external) | ~€1/year |
| Route 53 Hosted Zones | ✅ Yes | $0.50/month |
| Route 53 DNS Queries | ✅ Yes | $0.40/million queries |
| ACM Certificates | ✅ Yes | **Free** |
| ALB (HTTPS) | ✅ Yes (Free Tier hours) | First 750 hours/month free |

**Total estimated cost**: ~€1/year (domain) + ~$0.50/month (hosted zone) = **~€7/year**

---

## Why Use a Custom Domain?

Using the AWS ALB hostname (e.g., `k8s-overcook-overcook-4c03cb3f56-1082818309.eu-central-1.elb.amazonaws.com`) has issues:
- The hostname **changes** every time the ALB is recreated
- Google OAuth requires **stable redirect URIs**
- Not user-friendly or memorable

A custom domain like `app.overcookied.de` provides:
- ✅ Stable URL for OAuth configuration
- ✅ Professional appearance
- ✅ HTTPS support with ACM certificates
- ✅ Easy DNS management

---

## Step 1: Acquire the Domain

### Option A: Register through IONOS (Recommended for .de domains)

**IONOS is recommended** because:
- ✅ Cheapest .de domains (~€1/year first year)
- ✅ Easy nameserver configuration
- ✅ Works perfectly with AWS Free Tier

**Steps to register at IONOS:**
1. Go to https://www.ionos.de/domains/de-domain
2. Search for `overcookied.de`
3. Add to cart and complete purchase
4. After purchase, you'll have access to DNS settings

### Other German Registrars

| Registrar | URL | Approximate Price |
|-----------|-----|-------------------|
| IONOS (1&1) | https://www.ionos.de/domains/de-domain | ~€1/year first year |
| Strato | https://www.strato.de/domains/ | ~€0.50/month |
| United Domains | https://www.united-domains.de/ | ~€12/year |
| Namecheap | https://www.namecheap.com/ | ~$10/year |

### ❌ Option B: Register through AWS Route 53 (NOT available for Free Tier)

> ⚠️ **Note**: Your AWS account is on Free Tier and doesn't support domain registration. Use IONOS or another registrar instead.

---

## Step 2: Create Hosted Zone in AWS

After acquiring the domain from IONOS, create a hosted zone in Route 53:

```powershell
# Create hosted zone
aws route53 create-hosted-zone --name overcookied.de --caller-reference "overcookied-$(Get-Date -Format 'yyyyMMddHHmmss')"
```

This will output nameservers like:
```
ns-123.awsdns-45.com
ns-678.awsdns-90.net
ns-1234.awsdns-56.co.uk
ns-987.awsdns-12.org
```

**Save these nameservers** - you'll need them in the next step!

---

## Step 3: Update Nameservers at IONOS

Configure IONOS to use AWS Route 53 for DNS:

1. Log in to IONOS at https://my.ionos.de
2. Go to **Domains & SSL** → **overcookied.de**
3. Click **DNS** or **Nameserver Settings**
4. Select **Use custom nameservers**
5. Enter the 4 AWS nameservers from Step 2:
   ```
   ns-123.awsdns-45.com
   ns-678.awsdns-90.net
   ns-1234.awsdns-56.co.uk
   ns-987.awsdns-12.org
   ```
6. Save changes

> ⏳ **DNS propagation can take 24-48 hours.** You can check progress with `nslookup overcookied.de`

---

## Step 4: Create ACM Certificate for HTTPS (FREE!)

AWS Certificate Manager provides **free** SSL/TLS certificates:

```powershell
# Request certificate (in the same region as your ALB)
aws acm request-certificate `
  --domain-name "overcookied.de" `
  --subject-alternative-names "*.overcookied.de" `
  --validation-method DNS `
  --region eu-central-1
```

Note the certificate ARN from the output.

Then add the DNS validation records:

```powershell
# Get validation records
aws acm describe-certificate --certificate-arn <YOUR_CERT_ARN> --query "Certificate.DomainValidationOptions"
```

Add the CNAME records to Route 53 for validation.

---

## Step 5: Create DNS Record for the Application

Create an A record (alias) pointing to the ALB:

```powershell
# Get the ALB hostname
$ALB_HOSTNAME = kubectl get ingress -n overcookied -o jsonpath='{.items[0].status.loadBalancer.ingress[0].hostname}'

# Get the hosted zone ID
$ZONE_ID = aws route53 list-hosted-zones --query "HostedZones[?Name=='overcookied.de.'].Id" --output text

# Create the record (using alias to ALB)
# Note: You'll need the ALB's zone ID for the alias target
```

Or use the Terraform configuration in `infra/eks/route53.tf`.

---

## Step 6: Update Kubernetes Ingress for HTTPS

Update `k8s/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: overcookied-ingress
  namespace: overcookied
  annotations:
    kubernetes.io/ingress.class: alb
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    # Add HTTPS configuration
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}, {"HTTPS": 443}]'
    alb.ingress.kubernetes.io/certificate-arn: <YOUR_ACM_CERTIFICATE_ARN>
    alb.ingress.kubernetes.io/ssl-redirect: '443'
spec:
  rules:
  - host: app.overcookied.de
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: frontend-service
            port:
              number: 3000
      - path: /auth
        pathType: Prefix
        backend:
          service:
            name: backend-service
            port:
              number: 8080
      - path: /ws
        pathType: Prefix
        backend:
          service:
            name: backend-service
            port:
              number: 8080
```

---

## Step 7: Update OAuth Configuration

### Update Google Cloud Console

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Navigate to APIs & Services → Credentials
3. Edit your OAuth 2.0 Client ID
4. Add authorized redirect URIs:
   - `https://app.overcookied.de/auth/google/callback`
5. Save

### Update Kubernetes ConfigMap

Update `k8s/backend/oauth-configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth-config
  namespace: overcookied
data:
  GOOGLE_REDIRECT_URL: "https://app.overcookied.de/auth/google/callback"
  FRONTEND_URL: "https://app.overcookied.de"
```

Apply:
```powershell
kubectl apply -f k8s/backend/oauth-configmap.yaml
kubectl rollout restart deployment/overcookied-backend -n overcookied
```

---

## Summary Checklist

- [x] Register domain `overcookied.de`
- [x] Create Route 53 hosted zone
- [x] Update nameservers at registrar
- [x] Wait for DNS propagation (24-48 hours)
- [x] Create ACM certificate
- [x] Validate ACM certificate via DNS
- [x] Create A record alias to ALB
- [x] Update Kubernetes Ingress with HTTPS
- [x] Update Google OAuth redirect URIs
- [x] Update OAuth ConfigMap
- [x] Test login at `https://overcookied.de`

---

## Re-Deployment After Cluster Destroy

When the EKS cluster is destroyed and recreated, follow these steps:

### What Persists (No action needed)
- Route 53 Hosted Zone and Nameserver settings
- ACM Certificate (already validated)
- DynamoDB tables
- AWS Secrets Manager (OAuth credentials)
- ECR images
- Google OAuth configuration (in Google Console)

### What Needs Recreation

1. **Deploy the cluster and application:**
   ```powershell
   # Deploy EKS
   cd infra/eks
   terraform apply
   
   # Update kubeconfig
   aws eks update-kubeconfig --region eu-central-1 --name overcookied-eks
   
   # Deploy application (this creates JWT secret automatically)
   .\scripts\deploy-app.ps1
   ```

2. **Update Route 53 DNS record (ALB hostname changes):**
   ```powershell
   .\scripts\update-oauth-config.ps1 -UpdateDNS -RestartBackend
   ```

3. **Verify everything works:**
   ```powershell
   # Test DNS resolution
   nslookup overcookied.de
   
   # Check pods
   kubectl get pods -n overcookied
   
   # Visit https://overcookied.de/login
   ```

### Key Resources Created by Scripts

| Resource | Created By | Persists? |
|----------|-----------|-----------|
| JWT Secret | `deploy-app.ps1` | No - recreated on deploy |
| OAuth ConfigMap | `deploy-app.ps1` | No - recreated with defaults |
| Route 53 A Record | `update-oauth-config.ps1 -UpdateDNS` | Yes - but needs update with new ALB |

---

## Troubleshooting

### DNS Not Resolving
```powershell
nslookup overcookied.de
```
Wait for propagation or check nameserver configuration.

### Certificate Not Validated
Check the ACM certificate status in AWS Console. Ensure DNS validation records are correct.

### OAuth Redirect Mismatch
Ensure the redirect URI in Google Console **exactly** matches what's in the ConfigMap.

### Invalid State Error
This means cookies aren't working properly. Check:
- HTTPS is configured correctly
- Cookie SameSite settings are correct for your domain
- Both pods are using the same OAuth state mechanism (cookie-based)
