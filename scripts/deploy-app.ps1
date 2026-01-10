# Deploy Application to Kubernetes

# Configuration
$NAMESPACE = "overcookied"
$PROJECT_ROOT = (Get-Item $PSScriptRoot).Parent.FullName

Write-Host "☸️  Deploying Application to Kubernetes..." -ForegroundColor Cyan
Write-Host ""

# Get Account ID for manifest updates
$ACCOUNT_ID = (aws sts get-caller-identity --query Account --output text)
Write-Host "📋 AWS Account ID: $ACCOUNT_ID" -ForegroundColor White
Write-Host ""

# Check if kubectl is configured
Write-Host "🔍 Checking kubectl configuration..." -ForegroundColor Yellow
kubectl cluster-info *>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ kubectl not configured. Run: aws eks update-kubeconfig --region eu-central-1 --name overcookied-eks" -ForegroundColor Red
    exit 1
}
Write-Host "✅ kubectl configured" -ForegroundColor Green
Write-Host ""

# Update manifests with Account ID
Write-Host "📝 Updating Kubernetes manifests with Account ID..." -ForegroundColor Yellow
$FILES_TO_UPDATE = @(
    "$PROJECT_ROOT\k8s\backend\serviceaccount.yaml",
    "$PROJECT_ROOT\k8s\backend\deployment.yaml",
    "$PROJECT_ROOT\k8s\frontend\deployment.yaml"
)

foreach ($file in $FILES_TO_UPDATE) {
    if (Test-Path $file) {
        (Get-Content $file) -replace 'ACCOUNT_ID', $ACCOUNT_ID | Set-Content $file
        Write-Host "  ✅ Updated $file" -ForegroundColor Green
    }
}
Write-Host ""

# Apply Namespace
Write-Host "🏗️  Creating namespace: $NAMESPACE" -ForegroundColor Yellow
kubectl apply -f "$PROJECT_ROOT\k8s\namespace.yaml"
Write-Host ""

# Deploy OAuth ConfigMap (with placeholder values initially)
Write-Host "🔑 Deploying OAuth ConfigMap..." -ForegroundColor Yellow
kubectl apply -f "$PROJECT_ROOT\k8s\backend\oauth-configmap.yaml"
Write-Host ""

# Deploy Backend
Write-Host "🔧 Deploying Backend..." -ForegroundColor Yellow
kubectl apply -f "$PROJECT_ROOT\k8s\backend\serviceaccount.yaml"
kubectl apply -f "$PROJECT_ROOT\k8s\backend\deployment.yaml"
kubectl apply -f "$PROJECT_ROOT\k8s\backend\service.yaml"
Write-Host "✅ Backend deployed" -ForegroundColor Green
Write-Host ""

# Deploy Frontend
Write-Host "🎨 Deploying Frontend..." -ForegroundColor Yellow
kubectl apply -f "$PROJECT_ROOT\k8s\frontend\deployment.yaml"
kubectl apply -f "$PROJECT_ROOT\k8s\frontend\service.yaml"
Write-Host "✅ Frontend deployed" -ForegroundColor Green
Write-Host ""

# Deploy Ingress
Write-Host "🌐 Deploying ALB Ingress..." -ForegroundColor Yellow
kubectl apply -f "$PROJECT_ROOT\k8s\ingress.yaml"
Write-Host "✅ Ingress created (ALB provisioning...)" -ForegroundColor Green
Write-Host ""

# Wait for pods
Write-Host "⏳ Waiting for pods to be ready..." -ForegroundColor Yellow
kubectl wait --for=condition=ready pod -l app=overcookied-backend -n $NAMESPACE --timeout=120s
kubectl wait --for=condition=ready pod -l app=overcookied-frontend -n $NAMESPACE --timeout=120s
Write-Host "✅ All pods are ready" -ForegroundColor Green
Write-Host ""

# Wait for ALB
Write-Host "⏳ Waiting for ALB to provision (this may take 3-5 minutes)..." -ForegroundColor Yellow
$ALB_URL = ""
$MAX_ATTEMPTS = 60
$ATTEMPT = 0

while (-not $ALB_URL -and $ATTEMPT -lt $MAX_ATTEMPTS) {
    $ATTEMPT++
    $ALB_URL = (kubectl get ingress overcookied-ingress -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>$null)
    if (-not $ALB_URL) {
        Write-Host "  Attempt $ATTEMPT/$MAX_ATTEMPTS - Waiting..." -ForegroundColor Gray
        Start-Sleep -Seconds 5
    }
}

if ($ALB_URL) {
    Write-Host "✅ ALB provisioned successfully" -ForegroundColor Green
    Write-Host ""
    
    # Update OAuth ConfigMap with actual ALB URL
    Write-Host "🔑 Updating OAuth configuration with ALB URL..." -ForegroundColor Yellow
    $REDIRECT_URL = "http://$ALB_URL/auth/google/callback"
    $FRONTEND_URL = "http://$ALB_URL"
    
    $oauthConfigMap = @"
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth-config
  namespace: $NAMESPACE
  labels:
    app: overcookied
    component: config
data:
  GOOGLE_REDIRECT_URL: "$REDIRECT_URL"
  FRONTEND_URL: "$FRONTEND_URL"
"@
    $oauthConfigMap | kubectl apply -f -
    Write-Host "✅ OAuth ConfigMap updated" -ForegroundColor Green
    
    # Restart backend to pick up new config
    Write-Host "🔄 Restarting backend pods to apply OAuth config..." -ForegroundColor Yellow
    kubectl rollout restart deployment/overcookied-backend -n $NAMESPACE
    kubectl rollout status deployment/overcookied-backend -n $NAMESPACE --timeout=120s
    Write-Host "✅ Backend restarted with new OAuth config" -ForegroundColor Green
    Write-Host ""
    
    Write-Host "🎉 Deployment Complete!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Application URL: http://$ALB_URL" -ForegroundColor Cyan
    Write-Host ""
    
    # Display Google OAuth configuration
    Write-Host "=" * 60 -ForegroundColor Cyan
    Write-Host "🔑 GOOGLE OAUTH CONSOLE CONFIGURATION" -ForegroundColor Yellow
    Write-Host "=" * 60 -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Update these in the Google Cloud Console:" -ForegroundColor White
    Write-Host ""
    Write-Host "Authorized JavaScript Origins:" -ForegroundColor Cyan
    Write-Host "   $FRONTEND_URL" -ForegroundColor White
    Write-Host ""
    Write-Host "Authorized Redirect URIs:" -ForegroundColor Cyan
    Write-Host "   $REDIRECT_URL" -ForegroundColor White
    Write-Host ""
    Write-Host "=" * 60 -ForegroundColor Cyan
    Write-Host ""
    
    Write-Host "Testing backend health..." -ForegroundColor Yellow
    Start-Sleep -Seconds 10  # Give ALB time to register targets
    try {
        $response = Invoke-WebRequest -Uri "http://$ALB_URL/health" -UseBasicParsing
        Write-Host "✅ Backend is healthy: $($response.Content)" -ForegroundColor Green
    } catch {
        Write-Host "⚠️  Backend health check failed (targets may still be registering)" -ForegroundColor Yellow
    }
    Write-Host ""
    Write-Host "Open in browser: http://$ALB_URL" -ForegroundColor Cyan
    Start-Process "http://$ALB_URL"
} else {
    Write-Host "❌ ALB provisioning timed out. Check AWS console or kubectl get ingress -n $NAMESPACE" -ForegroundColor Red
}
