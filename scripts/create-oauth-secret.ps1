# Create Google OAuth Secret in AWS Secrets Manager

# Configuration
$SECRET_NAME = "overcookied/google-oauth"
$REGION = "eu-central-1"

Write-Host "🔐 Creating Google OAuth Secret in AWS Secrets Manager..." -ForegroundColor Cyan
Write-Host ""

# Prompt for credentials
Write-Host "Please enter your Google OAuth credentials:" -ForegroundColor Yellow
$GOOGLE_CLIENT_ID = Read-Host "Google Client ID"
$GOOGLE_CLIENT_SECRET = Read-Host "Google Client Secret" -AsSecureString
$GOOGLE_CLIENT_SECRET_PLAIN = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($GOOGLE_CLIENT_SECRET)
)

# Create secret JSON
$SECRET_VALUE = @{
    client_id = $GOOGLE_CLIENT_ID
    client_secret = $GOOGLE_CLIENT_SECRET_PLAIN
} | ConvertTo-Json -Compress

# Create secret
Write-Host ""
Write-Host "📝 Creating secret: $SECRET_NAME" -ForegroundColor Yellow
try {
    aws secretsmanager create-secret `
        --name $SECRET_NAME `
        --description "Google OAuth Credentials for Overcookied" `
        --secret-string $SECRET_VALUE `
        --region $REGION
    Write-Host "✅ Secret created successfully" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Secret might already exist. Updating instead..." -ForegroundColor Yellow
    aws secretsmanager update-secret `
        --secret-id $SECRET_NAME `
        --secret-string $SECRET_VALUE `
        --region $REGION
    Write-Host "✅ Secret updated successfully" -ForegroundColor Green
}

Write-Host ""
Write-Host "✨ Google OAuth secret is ready!" -ForegroundColor Green
$ACCOUNT_ID = (aws sts get-caller-identity --query Account --output text)
Write-Host "Secret ARN: arn:aws:secretsmanager:${REGION}:${ACCOUNT_ID}:secret:$SECRET_NAME" -ForegroundColor White
