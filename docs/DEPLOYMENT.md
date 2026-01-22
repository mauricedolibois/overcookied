# Overcookied - EKS Deployment Guide

This guide walks you through deploying the Overcookied game application to AWS EKS using the 2-layer Terraform architecture.

## Prerequisites

✅ **AWS CLI** (v2.x) - configured with credentials  
✅ **Terraform** (v1.9+)  
✅ **kubectl** (v1.30+)  
✅ **Docker** - for building container images  
✅ **helm** (v3.16+) - for AWS Load Balancer Controller  
✅ **Go** (1.24.9+) - for building backend  
✅ **Node.js** (20.x) - for building frontend

### Verify Prerequisites

```powershell
aws --version
terraform --version
kubectl version --client
docker --version
helm version
go version
node --version
```

## Architecture Overview

The deployment uses a **2-layer Terraform architecture**:

- **Base Layer** (`infra/base`): Persistent infrastructure
  - VPC with public/private subnets across 3 AZs
  - ECR repositories (backend & frontend)
  - **Destroyed manually only** - survives cluster recreations

- **EKS Layer** (`infra/eks`): Ephemeral cluster infrastructure
  - EKS cluster with managed node groups
  - ElastiCache (Valkey 8.0) for distributed matchmaking
  - ALB Ingress Controller via Helm
  - DynamoDB access via IRSA (IAM Roles for Service Accounts)
  - **Can be destroyed and recreated** to save costs

- **Kubernetes** (`k8s/`): Application manifests
  - Backend deployment (2 replicas), service, HPA
  - Frontend deployment (1 replica), service
  - ALB ingress with HTTPS redirect
  - ConfigMaps for OAuth and Redis config

## Cost Estimate

**Base Layer** (persistent): ~€20-30/month
- VPC, subnets, NAT Gateway (1 per AZ: €32-48/month for 3 AZs = avoid!)
- ECR repositories: minimal

**EKS Layer** (ephemeral): ~€50-70/month
- EKS cluster: €0.10/hour (~€75/month)
- EC2 nodes (2x t3.medium): ~€30/month
- ElastiCache (t3.small, 1GB): ~€25/month
- ALB: ~€16/month

**Total Cluster**: ~€70-90/month

## Phase 0: Bootstrap (ONE-TIME SETUP)

### Step 1: Create S3 Bucket and DynamoDB Table for Terraform State

```powershell
# Create S3 bucket for Terraform state
aws s3api create-bucket `
  --bucket overcookied-terraform-state `
  --region eu-central-1 `
  --create-bucket-configuration LocationConstraint=eu-central-1

# Enable versioning
aws s3api put-bucket-versioning `
  --bucket overcookied-terraform-state `
  --versioning-configuration Status=Enabled

# Create DynamoDB table for state locking
aws dynamodb create-table `
  --table-name terraform-state-lock `
  --attribute-definitions AttributeName=LockID,AttributeType=S `
  --key-schema AttributeName=LockID,KeyType=HASH `
  --billing-mode PAY_PER_REQUEST `
  --region eu-central-1
```

### Step 2: Create Google OAuth Secret in AWS Secrets Manager

```powershell
# Replace with your actual Google OAuth credentials
aws secretsmanager create-secret `
  --name overcookied/google-oauth `
  --description "Google OAuth Credentials for Overcookied" `
  --secret-string '{\"client_id\":\"YOUR_CLIENT_ID\",\"client_secret\":\"YOUR_CLIENT_SECRET\"}' `
  --region eu-central-1
```

### Step 3: Get Your AWS Account ID

```powershell
$ACCOUNT_ID = (aws sts get-caller-identity --query Account --output text)
Write-Host "Your AWS Account ID: $ACCOUNT_ID"
```

**Save this Account ID** - you'll need it for Kubernetes manifests.

---

## Phase 1: Deploy Base Infrastructure

### Step 1: Navigate to Base Layer

```powershell
cd infra\base
```

### Step 2: Initialize Terraform

```powershell
terraform init
```

### Step 3: Review and Apply

```powershell
# Review the plan
terraform plan -out=base.tfplan

# Apply
terraform apply base.tfplan
```

**Expected Duration**: ~2-3 minutes

### Step 4: Save Outputs

```powershell
# Save outputs for later use
terraform output -json | Out-File -FilePath base_outputs.json

# Get ECR URLs
terraform output ecr_backend_url
terraform output ecr_frontend_url
```

---

## Phase 2: Build and Push Container Images

### Step 1: Login to ECR

```powershell
$ACCOUNT_ID = (aws sts get-caller-identity --query Account --output text)
aws ecr get-login-password --region eu-central-1 | docker login --username AWS --password-stdin "$ACCOUNT_ID.dkr.ecr.eu-central-1.amazonaws.com"
```

### Step 2: Build and Push Backend

```powershell
cd ..\..\backend

# Build
docker build -t overcookied-backend:latest .

# Tag
$ECR_BACKEND = (terraform -chdir=..\infra\base output -raw ecr_backend_url)
docker tag overcookied-backend:latest $ECR_BACKEND`:latest

# Push
docker push $ECR_BACKEND`:latest
```

### Step 3: Build and Push Frontend

```powershell
cd ..\frontend

# Build
docker build -t overcookied-frontend:latest .

# Tag
$ECR_FRONTEND = (terraform -chdir=..\infra\base output -raw ecr_frontend_url)
docker tag overcookied-frontend:latest $ECR_FRONTEND`:latest

# Push
docker push $ECR_FRONTEND`:latest
```

---

## Phase 3: Deploy EKS Cluster

### Step 1: Navigate to EKS Layer

```powershell
cd ..\infra\eks
```

### Step 2: Initialize Terraform

```powershell
terraform init
```

### Step 3: Review and Apply

```powershell
# Review the plan
terraform plan -out=eks.tfplan

# Apply (this takes ~15-20 minutes)
terraform apply eks.tfplan
```

**Expected Duration**: ~15-20 minutes (EKS cluster creation)

### Step 4: Configure kubectl

```powershell
aws eks update-kubeconfig --region eu-central-1 --name overcookied-eks

# Verify
kubectl get nodes
```

Expected output: 2 nodes in `Ready` status

### Step 5: Verify AWS Load Balancer Controller

```powershell
kubectl get deployment -n kube-system aws-load-balancer-controller
```

Expected: 1/1 Ready

---

## Phase 4: Deploy Application to Kubernetes

### Step 1: Update Kubernetes Manifests with Account ID

**Edit the following files** and replace `ACCOUNT_ID` with your AWS Account ID:

- `k8s\backend\serviceaccount.yaml` - Line 6
- `k8s\backend\deployment.yaml` - Line 22
- `k8s\frontend\deployment.yaml` - Line 22

```powershell
# Get your Account ID
$ACCOUNT_ID = (aws sts get-caller-identity --query Account --output text)
Write-Host "Replace ACCOUNT_ID with: $ACCOUNT_ID"
```

### Step 2: Apply Namespace

```powershell
cd ..\..
kubectl apply -f k8s\namespace.yaml
```

### Step 3: Deploy Backend

```powershell
kubectl apply -f k8s\backend\serviceaccount.yaml
kubectl apply -f k8s\backend\deployment.yaml
kubectl apply -f k8s\backend\service.yaml
```

### Step 4: Deploy Frontend

```powershell
kubectl apply -f k8s\frontend\deployment.yaml
kubectl apply -f k8s\frontend\service.yaml
```

### Step 5: Deploy Ingress (ALB)

```powershell
kubectl apply -f k8s\ingress.yaml
```

### Step 6: Wait for ALB Provisioning (~3-5 minutes)

```powershell
kubectl get ingress -n overcookied -w
```

Wait until the `ADDRESS` column shows an ALB DNS name (e.g., `k8s-overcook-xxxxxxxx-123456789.eu-central-1.elb.amazonaws.com`)

Press `Ctrl+C` to exit watch mode.

### Step 7: Get Application URL

```powershell
$ALB_URL = (kubectl get ingress overcookied-ingress -n overcookied -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
Write-Host "Application URL: http://$ALB_URL"
```

### Step 8: Test Application

```powershell
# Test backend health
curl http://$ALB_URL/health

# Expected: {"message":"Backend is running","status":"healthy"}

# Open in browser
Start-Process "http://$ALB_URL"
```

---

## Phase 5: Optional - Deploy HPA (Horizontal Pod Autoscaler)

```powershell
kubectl apply -f k8s\backend\hpa.yaml
```

Verify:

```powershell
kubectl get hpa -n overcookied
```

---

## Phase 6: Destroy EKS (without destroying Base)

### Step 1: Delete Kubernetes Resources First

**IMPORTANT**: Delete Ingress first to avoid dangling ENIs/Load Balancers

```powershell
kubectl delete ingress overcookied-ingress -n overcookied
Start-Sleep -Seconds 120  # Wait 2 minutes for ALB cleanup

kubectl delete all --all -n overcookied
kubectl delete namespace overcookied
```

### Step 2: Terraform Destroy EKS

```powershell
cd infra\eks
terraform destroy
```

Type `yes` to confirm.

### Step 3: Verify Base Layer is Intact

```powershell
cd ..\base
terraform state list
```

VPC, Subnets, and ECR should still be present.

---

## Phase 7: Recreate EKS

Since the Base Layer (VPC, ECR) still exists, recreating EKS is fast:

```powershell
cd ..\eks
terraform apply
```

Then repeat **Phase 4** steps to redeploy the application.

---

## Monitoring and Troubleshooting

### View Pod Logs

```powershell
# Backend logs
kubectl logs -n overcookied -l app=overcookied-backend --tail=100 -f

# Frontend logs
kubectl logs -n overcookied -l app=overcookied-frontend --tail=100 -f
```

### Check Pod Status

```powershell
kubectl get pods -n overcookied
kubectl describe pod <pod-name> -n overcookied
```

### Check ALB Target Health

```powershell
aws elbv2 describe-target-health `
  --target-group-arn (aws elbv2 describe-target-groups --query 'TargetGroups[?contains(TargetGroupName, `overcookied`)].TargetGroupArn' --output text)
```

### Check IRSA (IAM Roles for Service Accounts)

```powershell
# Verify service account annotation
kubectl get sa backend-sa -n overcookied -o yaml

# Check pod environment for AWS credentials
kubectl exec -it <backend-pod-name> -n overcookied -- env | Select-String AWS
```

---

## Cost Optimization Tips

✅ **Stop EKS when not in use**: `terraform destroy` in `infra/eks` (Base layer remains)
✅ **Use Spot Instances**: Update `capacity_type = "SPOT"` in `eks.tf` node group
✅ **Reduce node count**: Set `node_desired_size = 1` in `terraform.tfvars`
✅ **Monitor CloudWatch Logs**: Set retention to 7 days instead of indefinite

---

## Security Best Practices

✅ **IRSA enabled**: No AWS credentials hardcoded in pods
✅ **Secrets Manager**: Google OAuth credentials stored securely
✅ **Security Groups**: Nodes only accessible via ALB
✅ **ECR Image Scanning**: Enabled for vulnerability detection
✅ **Pod Resource Limits**: Prevents resource exhaustion

---

## Next Steps

1. **Domain & TLS**: Add Route53 hosted zone and ACM certificate for HTTPS
2. **CI/CD**: Automate deployments with GitHub Actions
3. **Monitoring**: Enable CloudWatch Container Insights for metrics
4. **Backup**: Enable DynamoDB Point-in-Time Recovery (PITR)
5. **Multi-Environment**: Duplicate setup for `dev`/`staging` environments

---

## Troubleshooting Common Issues

### Issue: ALB not creating

**Solution**: Check AWS Load Balancer Controller logs

```powershell
kubectl logs -n kube-system deployment/aws-load-balancer-controller
```

### Issue: Backend pods can't access DynamoDB

**Solution**: Verify IRSA setup

```powershell
# Check service account annotation
kubectl get sa backend-sa -n overcookied -o jsonpath='{.metadata.annotations}'

# Check pod logs for AWS errors
kubectl logs -n overcookied -l app=overcookied-backend | Select-String -Pattern "AWS\|DynamoDB"
```

### Issue: Pods stuck in `ImagePullBackOff`

**Solution**: Verify ECR permissions for node IAM role

```powershell
# Check if images exist in ECR
aws ecr list-images --repository-name overcookied-backend --region eu-central-1
```

---

## Support

For issues or questions:
- Check CloudWatch Logs: `/aws/eks/overcookied-eks/cluster`
- Review Terraform outputs: `terraform output -json`
- Verify WebSocket connectivity in browser DevTools (Network tab)
