<#
.SYNOPSIS
    Updates the OAuth ConfigMap with the custom domain URL after deployment
.DESCRIPTION
    This script updates the oauth-config ConfigMap with the correct redirect URLs
    for the custom domain (overcookied.de). It also updates Route 53 if needed.
    Run this after the ALB is provisioned.
#>

param(
    [string]$Namespace = "overcookied",
    [string]$Domain = "overcookied.de",
    [switch]$RestartBackend,
    [switch]$UpdateDNS
)

$ErrorActionPreference = "Stop"

Write-Host "`n🔧 Updating OAuth Configuration for $Domain..." -ForegroundColor Cyan

# Wait for ALB to be provisioned
Write-Host "⏳ Waiting for ALB hostname..." -ForegroundColor Yellow
$maxAttempts = 30
$attempt = 0
$albHostname = ""

while ($attempt -lt $maxAttempts) {
    $attempt++
    $albHostname = kubectl get ingress -n $Namespace overcookied-ingress -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>$null
    
    if ($albHostname -and $albHostname -ne "") {
        Write-Host "✅ ALB hostname found: $albHostname" -ForegroundColor Green
        break
    }
    
    Write-Host "   Attempt $attempt/$maxAttempts - ALB not ready yet..." -ForegroundColor Gray
    Start-Sleep -Seconds 10
}

if (-not $albHostname -or $albHostname -eq "") {
    Write-Host "❌ Failed to get ALB hostname after $maxAttempts attempts" -ForegroundColor Red
    exit 1
}

# Update Route 53 DNS record if requested
if ($UpdateDNS) {
    Write-Host "`n🌐 Updating Route 53 DNS record..." -ForegroundColor Yellow
    
    # Get the hosted zone ID
    $zoneId = aws route53 list-hosted-zones --query "HostedZones[?Name=='$Domain.'].Id" --output text
    if (-not $zoneId) {
        Write-Host "❌ Hosted zone for $Domain not found" -ForegroundColor Red
        exit 1
    }
    $zoneId = $zoneId -replace '/hostedzone/', ''
    Write-Host "   Hosted Zone ID: $zoneId" -ForegroundColor Gray
    
    # Create the change batch JSON
    $changeBatch = @"
{
  "Changes": [
    {
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "$Domain",
        "Type": "A",
        "AliasTarget": {
          "HostedZoneId": "Z215JYRZR1TBD5",
          "DNSName": "$albHostname",
          "EvaluateTargetHealth": true
        }
      }
    }
  ]
}
"@
    
    $tempFile = [System.IO.Path]::GetTempFileName()
    $changeBatch | Out-File -FilePath $tempFile -Encoding UTF8
    
    aws route53 change-resource-record-sets --hosted-zone-id $zoneId --change-batch file://$tempFile
    Remove-Item $tempFile
    
    Write-Host "✅ DNS record updated for $Domain" -ForegroundColor Green
}

# Construct the URLs with HTTPS
$baseUrl = "https://$Domain"
$redirectUrl = "$baseUrl/auth/google/callback"

Write-Host "`n📝 OAuth URLs:" -ForegroundColor Cyan
Write-Host "   Base URL: $baseUrl"
Write-Host "   Redirect URL: $redirectUrl"

# Update the ConfigMap
Write-Host "`n🔄 Updating oauth-config ConfigMap..." -ForegroundColor Yellow

$configMapYaml = @"
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth-config
  namespace: $Namespace
  labels:
    app: overcookied
    component: config
data:
  GOOGLE_REDIRECT_URL: "$redirectUrl"
  FRONTEND_URL: "$baseUrl"
"@

# Apply the ConfigMap
$configMapYaml | kubectl apply -f -

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ ConfigMap updated successfully" -ForegroundColor Green
} else {
    Write-Host "❌ Failed to update ConfigMap" -ForegroundColor Red
    exit 1
}

# Restart backend pods to pick up new config
if ($RestartBackend) {
    Write-Host "`n🔄 Restarting backend pods..." -ForegroundColor Yellow
    kubectl rollout restart deployment/overcookied-backend -n $Namespace
    
    Write-Host "⏳ Waiting for rollout to complete..." -ForegroundColor Yellow
    kubectl rollout status deployment/overcookied-backend -n $Namespace --timeout=120s
    
    Write-Host "✅ Backend restarted successfully" -ForegroundColor Green
}

Write-Host "`n" + "=" * 60 -ForegroundColor Cyan
Write-Host "🔑 GOOGLE OAUTH CONSOLE CONFIGURATION" -ForegroundColor Yellow
Write-Host "=" * 60 -ForegroundColor Cyan
Write-Host "`nUpdate these in the Google Cloud Console:" -ForegroundColor White
Write-Host "`nAutorized JavaScript Origins:" -ForegroundColor Cyan
Write-Host "   $baseUrl" -ForegroundColor White
Write-Host "`nAutorized Redirect URIs:" -ForegroundColor Cyan
Write-Host "   $redirectUrl" -ForegroundColor White
Write-Host "`n" + "=" * 60 -ForegroundColor Cyan

Write-Host "`n✅ OAuth configuration complete!" -ForegroundColor Green
Write-Host "⚠️  Remember to update the Google OAuth Console with the URLs above!" -ForegroundColor Yellow
