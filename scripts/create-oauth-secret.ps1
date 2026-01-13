# Create Google OAuth Secret and JWT Secret in AWS Secrets Manager

# Configuration
$OAUTH_SECRET_NAME = "overcookied/google-oauth"
$JWT_SECRET_NAME = "overcookied/jwt-secret"
$REGION = "eu-central-1"

Write-Host "🔐 Creating Secrets in AWS Secrets Manager..." -ForegroundColor Cyan
Write-Host ""

# =============================================================================
# Part 1: Google OAuth Secret
# =============================================================================
Write-Host "📝 Part 1: Google OAuth Credentials" -ForegroundColor Yellow
Write-Host "Please enter your Google OAuth credentials:" -ForegroundColor White
$GOOGLE_CLIENT_ID = Read-Host "Google Client ID"
$GOOGLE_CLIENT_SECRET = Read-Host "Google Client Secret" -AsSecureString
$GOOGLE_CLIENT_SECRET_PLAIN = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($GOOGLE_CLIENT_SECRET)
)

# Create secret JSON
$OAUTH_SECRET_VALUE = @{
    client_id = $GOOGLE_CLIENT_ID
    client_secret = $GOOGLE_CLIENT_SECRET_PLAIN
} | ConvertTo-Json -Compress

# Create/Update OAuth secret
Write-Host ""
Write-Host "📝 Creating secret: $OAUTH_SECRET_NAME" -ForegroundColor Yellow
try {
    aws secretsmanager create-secret `
        --name $OAUTH_SECRET_NAME `
        --description "Google OAuth Credentials for Overcookied" `
        --secret-string $OAUTH_SECRET_VALUE `
        --region $REGION 2>$null
    Write-Host "✅ OAuth secret created successfully" -ForegroundColor Green
} catch {
    Write-Host "⚠️  OAuth secret might already exist. Updating instead..." -ForegroundColor Yellow
    aws secretsmanager update-secret `
        --secret-id $OAUTH_SECRET_NAME `
        --secret-string $OAUTH_SECRET_VALUE `
        --region $REGION
    Write-Host "✅ OAuth secret updated successfully" -ForegroundColor Green
}

# =============================================================================
# Part 2: JWT Secret
# =============================================================================
Write-Host ""
Write-Host "📝 Part 2: JWT Secret for Token Signing" -ForegroundColor Yellow

# Check if JWT secret already exists
$existingJwtSecret = aws secretsmanager describe-secret --secret-id $JWT_SECRET_NAME --region $REGION 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ JWT secret already exists in Secrets Manager" -ForegroundColor Green
} else {
    # Generate a secure random JWT secret
    $JWT_SECRET_VALUE = -join ((65..90) + (97..122) + (48..57) | Get-Random -Count 64 | ForEach-Object {[char]$_})
    
    $JWT_SECRET_JSON = @{
        jwt_secret = $JWT_SECRET_VALUE
    } | ConvertTo-Json -Compress
    
    Write-Host "📝 Creating secret: $JWT_SECRET_NAME" -ForegroundColor Yellow
    aws secretsmanager create-secret `
        --name $JWT_SECRET_NAME `
        --description "JWT Secret for Overcookied token signing (shared across all pods)" `
        --secret-string $JWT_SECRET_JSON `
        --region $REGION
    Write-Host "✅ JWT secret created successfully" -ForegroundColor Green
}

# =============================================================================
# Summary
# =============================================================================
Write-Host ""
Write-Host "=" * 60 -ForegroundColor Cyan
Write-Host "✨ All secrets are ready!" -ForegroundColor Green
Write-Host "=" * 60 -ForegroundColor Cyan
$ACCOUNT_ID = (aws sts get-caller-identity --query Account --output text)
Write-Host ""
Write-Host "Secrets created:" -ForegroundColor White
Write-Host "  📌 OAuth: arn:aws:secretsmanager:${REGION}:${ACCOUNT_ID}:secret:$OAUTH_SECRET_NAME" -ForegroundColor Gray
Write-Host "  📌 JWT:   arn:aws:secretsmanager:${REGION}:${ACCOUNT_ID}:secret:$JWT_SECRET_NAME" -ForegroundColor Gray
Write-Host ""
Write-Host "⚠️  Note: Backend currently uses Kubernetes Secret for JWT." -ForegroundColor Yellow
Write-Host "   The AWS Secrets Manager JWT secret is a backup for persistence." -ForegroundColor Yellow
