# Destroy EKS Infrastructure (keeps Base layer intact)

# Configuration
$NAMESPACE = "overcookied"
$PROJECT_ROOT = (Get-Item $PSScriptRoot).Parent.FullName

Write-Host "🗑️  Destroying EKS Infrastructure..." -ForegroundColor Cyan
Write-Host ""

Write-Host "⚠️  WARNING: This will destroy the EKS cluster but keep Base infrastructure (VPC, ECR)" -ForegroundColor Yellow
$confirm = Read-Host "Are you sure? (type 'yes' to confirm)"

if ($confirm -ne 'yes') {
    Write-Host "❌ Aborted" -ForegroundColor Red
    exit 0
}

Write-Host ""
Write-Host "🧹 Step 1: Cleaning up Kubernetes resources..." -ForegroundColor Yellow

# Check if kubectl is configured
kubectl cluster-info *>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "  Deleting Ingress (to remove ALB)..." -ForegroundColor Yellow
    kubectl delete ingress overcookied-ingress -n $NAMESPACE 2>$null
    
    Write-Host "  Waiting 2 minutes for ALB cleanup..." -ForegroundColor Yellow
    Start-Sleep -Seconds 120
    
    Write-Host "  Deleting all resources in namespace..." -ForegroundColor Yellow
    kubectl delete all --all -n $NAMESPACE 2>$null
    
    Write-Host "  Deleting namespace..." -ForegroundColor Yellow
    kubectl delete namespace $NAMESPACE 2>$null
    
    Write-Host "✅ Kubernetes resources deleted" -ForegroundColor Green
} else {
    Write-Host "⚠️  kubectl not configured, skipping Kubernetes cleanup" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "🗑️  Step 2: Destroying EKS Terraform stack..." -ForegroundColor Yellow
Push-Location "$PROJECT_ROOT\infra\eks"
terraform destroy -auto-approve
Pop-Location

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ EKS infrastructure destroyed" -ForegroundColor Green
} else {
    Write-Host "❌ Terraform destroy failed" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✅ Verifying Base layer is intact..." -ForegroundColor Yellow
Push-Location "$PROJECT_ROOT\infra\base"
$resources = terraform state list
Pop-Location
if ($resources) {
    Write-Host "✅ Base layer resources still present:" -ForegroundColor Green
    $resources | ForEach-Object { Write-Host "  - $_" -ForegroundColor White }
} else {
    Write-Host "⚠️  Could not verify Base layer" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "✨ EKS destroyed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Base infrastructure (VPC, ECR) is still running." -ForegroundColor Cyan
Write-Host "To recreate EKS: cd infra\eks && terraform apply" -ForegroundColor White
