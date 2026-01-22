# Bootstrap Terraform State (ONE-TIME SETUP)
# Run this BEFORE any terraform init/apply commands

# Configuration
$BUCKET_NAME = "overcookied-terraform-state"
$TABLE_NAME = "terraform-state-lock"
$REGION = "eu-central-1"
$PROJECT_ROOT = (Get-Item $PSScriptRoot).Parent.FullName

Write-Host "🚀 Bootstrapping Terraform Remote State Infrastructure..." -ForegroundColor Cyan
Write-Host ""

# Create S3 Bucket
Write-Host "📦 Creating S3 bucket: $BUCKET_NAME" -ForegroundColor Yellow
try {
    aws s3api create-bucket `
        --bucket $BUCKET_NAME `
        --region $REGION `
        --create-bucket-configuration LocationConstraint=$REGION
    Write-Host "✅ S3 bucket created successfully" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Bucket might already exist or error occurred: $_" -ForegroundColor Yellow
}

# Enable versioning
Write-Host "🔄 Enabling versioning on S3 bucket..." -ForegroundColor Yellow
aws s3api put-bucket-versioning `
    --bucket $BUCKET_NAME `
    --versioning-configuration Status=Enabled
Write-Host "✅ Versioning enabled" -ForegroundColor Green

# Create DynamoDB table for state locking
Write-Host "🔒 Creating DynamoDB table: $TABLE_NAME" -ForegroundColor Yellow
try {
    aws dynamodb create-table `
        --table-name $TABLE_NAME `
        --attribute-definitions AttributeName=LockID,AttributeType=S `
        --key-schema AttributeName=LockID,KeyType=HASH `
        --billing-mode PAY_PER_REQUEST `
        --region $REGION
    Write-Host "✅ DynamoDB table created successfully" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Table might already exist or error occurred: $_" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "✨ Bootstrap complete!" -ForegroundColor Green
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Create Google OAuth secret: .\scripts\create-oauth-secret.ps1" -ForegroundColor White
Write-Host "  2. Deploy base infrastructure: cd infra\base && terraform init && terraform apply" -ForegroundColor White
