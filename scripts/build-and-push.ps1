# Build and Push Container Images to ECR

# Configuration
$REGION = "eu-central-1"
$PROJECT_ROOT = (Get-Item $PSScriptRoot).Parent.FullName

Write-Host "🐳 Building and Pushing Container Images to ECR..." -ForegroundColor Cyan
Write-Host ""

# Get Account ID
$ACCOUNT_ID = (aws sts get-caller-identity --query Account --output text)
Write-Host "📋 AWS Account ID: $ACCOUNT_ID" -ForegroundColor White

# Get ECR URLs from Terraform
Write-Host "🔍 Getting ECR repository URLs from Terraform..." -ForegroundColor Yellow
Push-Location "$PROJECT_ROOT\infra\base"
$ECR_BACKEND = (terraform output -raw ecr_backend_url)
$ECR_FRONTEND = (terraform output -raw ecr_frontend_url)
Pop-Location

if (-not $ECR_BACKEND -or -not $ECR_FRONTEND) {
    Write-Host "❌ Error: Could not get ECR URLs. Make sure Base layer is deployed." -ForegroundColor Red
    Write-Host "  Try: cd $PROJECT_ROOT\infra\base && terraform output" -ForegroundColor Yellow
    exit 1
}

Write-Host "  Backend ECR:  $ECR_BACKEND" -ForegroundColor White
Write-Host "  Frontend ECR: $ECR_FRONTEND" -ForegroundColor White
Write-Host ""

# Login to ECR
Write-Host "🔑 Logging in to Amazon ECR..." -ForegroundColor Yellow
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin "$ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com"
Write-Host "✅ ECR login successful" -ForegroundColor Green
Write-Host ""

# Build and push Backend
Write-Host "🔨 Building Backend Docker image..." -ForegroundColor Yellow
docker build -t overcookied-backend:latest "$PROJECT_ROOT\backend"
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Backend build failed" -ForegroundColor Red
    exit 1
}

Write-Host "🏷️  Tagging Backend image..." -ForegroundColor Yellow
docker tag overcookied-backend:latest "${ECR_BACKEND}:latest"

Write-Host "📤 Pushing Backend image to ECR..." -ForegroundColor Yellow
docker push "${ECR_BACKEND}:latest"
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Backend push failed" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Backend image pushed successfully" -ForegroundColor Green
Write-Host ""

# Build and push Frontend
Write-Host "🔨 Building Frontend Docker image..." -ForegroundColor Yellow
docker build -t overcookied-frontend:latest "$PROJECT_ROOT\frontend"
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Frontend build failed" -ForegroundColor Red
    exit 1
}

Write-Host "🏷️  Tagging Frontend image..." -ForegroundColor Yellow
docker tag overcookied-frontend:latest "${ECR_FRONTEND}:latest"

Write-Host "📤 Pushing Frontend image to ECR..." -ForegroundColor Yellow
docker push "${ECR_FRONTEND}:latest"
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Frontend push failed" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Frontend image pushed successfully" -ForegroundColor Green
Write-Host ""

Write-Host "✨ All images built and pushed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Next step:" -ForegroundColor Cyan
Write-Host "  Deploy EKS cluster: cd infra\eks && terraform init && terraform apply" -ForegroundColor White
