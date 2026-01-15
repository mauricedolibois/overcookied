# Create EKS Cluster and Deploy Application
# This script runs all steps needed to create a new EKS cluster and deploy Overcookied

param(
    [switch]$SkipTerraform,
    [switch]$SkipDeploy
)

$ErrorActionPreference = "Stop"
$PROJECT_ROOT = (Get-Item $PSScriptRoot).Parent.FullName

Write-Host ""
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host "🚀 OVERCOOKIED - CREATE CLUSTER & DEPLOY" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""

# =============================================================================
# Step 1: Create EKS Cluster with Terraform
# =============================================================================
if (-not $SkipTerraform) {
    Write-Host "📦 Step 1/4: Creating EKS Cluster with Terraform..." -ForegroundColor Yellow
    Write-Host "   This takes ~15-20 minutes. Please wait..." -ForegroundColor Gray
    Write-Host ""
    
    Push-Location "$PROJECT_ROOT\infra\eks"
    
    try {
        # Initialize Terraform
        Write-Host "   🔧 Initializing Terraform..." -ForegroundColor Cyan
        terraform init -input=false
        if ($LASTEXITCODE -ne 0) { throw "Terraform init failed" }
        
        # Apply Terraform
        Write-Host "   🏗️  Applying Terraform (EKS Cluster)..." -ForegroundColor Cyan
        terraform apply -auto-approve -input=false
        if ($LASTEXITCODE -ne 0) { throw "Terraform apply failed" }
        
        Write-Host "   ✅ EKS Cluster created successfully!" -ForegroundColor Green
    }
    catch {
        Write-Host "   ❌ Terraform failed: $_" -ForegroundColor Red
        Pop-Location
        exit 1
    }
    
    Pop-Location
    Write-Host ""
} else {
    Write-Host "📦 Step 1/4: Skipping Terraform (--SkipTerraform)" -ForegroundColor Gray
}

# =============================================================================
# Step 2: Configure kubectl
# =============================================================================
Write-Host "🔑 Step 2/4: Configuring kubectl..." -ForegroundColor Yellow

aws eks update-kubeconfig --region eu-central-1 --name overcookied-eks
if ($LASTEXITCODE -ne 0) {
    Write-Host "   ❌ Failed to configure kubectl" -ForegroundColor Red
    exit 1
}

# Verify connection
kubectl cluster-info *>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "   ❌ kubectl connection failed" -ForegroundColor Red
    exit 1
}

Write-Host "   ✅ kubectl configured and connected!" -ForegroundColor Green
Write-Host ""

# =============================================================================
# Step 3: Deploy Application
# =============================================================================
if (-not $SkipDeploy) {
    Write-Host "🎮 Step 3/4: Deploying Application..." -ForegroundColor Yellow
    Write-Host ""
    
    Push-Location $PROJECT_ROOT
    
    try {
        & "$PROJECT_ROOT\scripts\deploy-app.ps1"
        if ($LASTEXITCODE -ne 0) { throw "deploy-app.ps1 failed" }
    }
    catch {
        Write-Host "   ❌ Deployment failed: $_" -ForegroundColor Red
        Pop-Location
        exit 1
    }
    
    Pop-Location
    Write-Host ""
} else {
    Write-Host "🎮 Step 3/4: Skipping Deploy (--SkipDeploy)" -ForegroundColor Gray
}

# =============================================================================
# Step 4: Configure Custom Domain & OAuth
# =============================================================================
Write-Host "🌐 Step 4/4: Configuring Custom Domain & OAuth..." -ForegroundColor Yellow
Write-Host ""

Push-Location $PROJECT_ROOT

try {
    & "$PROJECT_ROOT\scripts\update-oauth-config.ps1" -UpdateDNS -RestartBackend
    if ($LASTEXITCODE -ne 0) { throw "update-oauth-config.ps1 failed" }
}
catch {
    Write-Host "   ❌ OAuth/DNS configuration failed: $_" -ForegroundColor Red
    Pop-Location
    exit 1
}

Pop-Location

# =============================================================================
# Complete!
# =============================================================================
Write-Host ""
Write-Host "=" * 70 -ForegroundColor Green
Write-Host "🎉 DEPLOYMENT COMPLETE!" -ForegroundColor Green
Write-Host "=" * 70 -ForegroundColor Green
Write-Host ""
Write-Host "Your application is now running at:" -ForegroundColor White
Write-Host ""
Write-Host "   🔗 https://overcookied.de" -ForegroundColor Cyan
Write-Host ""
Write-Host "Verify:" -ForegroundColor White
Write-Host "   kubectl get pods -n overcookied" -ForegroundColor Gray
Write-Host "   curl https://overcookied.de/health" -ForegroundColor Gray
Write-Host ""

# Open in browser
$openBrowser = Read-Host "Open in browser? (Y/n)"
if ($openBrowser -ne "n" -and $openBrowser -ne "N") {
    Start-Process "https://overcookied.de"
}
