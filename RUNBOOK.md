# 🚀 Overcookied EKS Deployment Runbook

Interaktives Step-by-Step Deployment Guide für AWS EKS.

---

## ⚙️ Phase 0: Bootstrap (One-Time Setup)

### ✅ Schritt 1: Terraform State Infrastructure erstellen

```powershell
cd C:\Users\mauri\Local Docs\overcookied\scripts
.\bootstrap-state.ps1
```

**Erstellt:**
- S3 Bucket: `overcookied-terraform-state`
- DynamoDB Table: `terraform-state-lock`

---

### ✅ Schritt 2: Google OAuth Secret speichern

```powershell
.\create-oauth-secret.ps1
```

**Eingabe erforderlich:**
- Google Client ID
- Google Client Secret

**Speichert in:** AWS Secrets Manager (`overcookied/google-oauth`)

---

### ✅ Schritt 3: DynamoDB Tabellen verifizieren

```powershell
aws dynamodb list-tables --region eu-central-1 | Select-String "CookieUsers|CookieGames"
```

**Erwartete Ausgabe:**
```
"CookieUsers",
"CookieGames"
```

> [!NOTE]
> Falls Tabellen nicht existieren, siehe `backend/aws_setup.md` für manuelle Erstellung.

---

## 🏗️ Phase 1: Deploy Base Infrastructure (VPC, ECR)

### Schritt 1: Navigate zu Base Layer

```powershell
cd ..\infra\base
```

### Schritt 2: Terraform initialisieren

```powershell
terraform init
```

### Schritt 3: Infrastructure Plan prüfen

```powershell
terraform plan
```

**Erwartete Ressourcen:**
- 1 VPC
- 3 Public Subnets
- 1 Internet Gateway
- 2 ECR Repositories

### Schritt 4: Infrastructure erstellen

```powershell
terraform apply
```

Typ `yes` zur Bestätigung.

**Dauer:** ~2-3 Minuten

### Schritt 5: Outputs anzeigen

```powershell
terraform output
```

**Wichtige Outputs:**
- `ecr_backend_url`
- `ecr_frontend_url`
- `vpc_id`

---

## 🐳 Phase 2: Build & Push Container Images

### Schritt 1: Container Images bauen und pushen

```powershell
cd ..\..
.\scripts\build-and-push.ps1
```

**Das Script führt aus:**
1. ECR Login
2. Backend Docker Build (~1-2 Min)
3. Backend Push zu ECR
4. Frontend Docker Build (~3-5 Min)
5. Frontend Push zu ECR

**Dauer:** ~5-10 Minuten

### Schritt 2: Images in ECR verifizieren

```powershell
aws ecr describe-images --repository-name overcookied-backend --region eu-central-1 --query 'imageDetails[0].imageTags'
aws ecr describe-images --repository-name overcookied-frontend --region eu-central-1 --query 'imageDetails[0].imageTags'
```

**Erwartete Ausgabe:** `["latest"]`

---

## ⚙️ Phase 3: Deploy EKS Cluster

### Schritt 1: Navigate zu EKS Layer

```powershell
cd infra\eks
```

### Schritt 2: Terraform initialisieren

```powershell
terraform init
```

### Schritt 3: EKS Plan prüfen

```powershell
terraform plan
```

**Erwartete Ressourcen:**
- EKS Cluster (v1.31)
- Managed Node Group (2x t3.medium)
- OIDC Provider
- IAM Roles (Backend Pod, ALB Controller)
- Security Groups
- Helm Release (AWS Load Balancer Controller)

### Schritt 4: EKS Cluster erstellen

```powershell
terraform apply
```

Typ `yes` zur Bestätigung.

> [!WARNING]
> **Dauer: 15-20 Minuten** ☕
> EKS Control Plane braucht Zeit zum Starten.

### Schritt 5: kubectl konfigurieren

```powershell
aws eks update-kubeconfig --region eu-central-1 --name overcookied-eks
```

### Schritt 6: Cluster Nodes verifizieren

```powershell
kubectl get nodes
```

**Erwartete Ausgabe:**
```
NAME                                           STATUS   ROLES    AGE   VERSION
ip-10-0-1-xxx.eu-central-1.compute.internal   Ready    <none>   2m    v1.31.x
ip-10-0-2-xxx.eu-central-1.compute.internal   Ready    <none>   2m    v1.31.x
```

### Schritt 7: AWS Load Balancer Controller prüfen

```powershell
kubectl get deployment -n kube-system aws-load-balancer-controller
```

**Erwartete Ausgabe:**
```
NAME                           READY   UP-TO-DATE   AVAILABLE   AGE
aws-load-balancer-controller   1/1     1            1           5m
```

---

## 🚀 Phase 4: Deploy Application zu Kubernetes

### Option A: Automatisches Deployment (Empfohlen)

```powershell
cd ..\..
.\scripts\deploy-app.ps1
```

**Das Script führt aus:**
1. Account ID in Manifests ersetzen
2. Namespace erstellen
3. Backend deployen (ServiceAccount, Deployment, Service)
4. Frontend deployen
5. ALB Ingress erstellen
6. Auf Pods warten
7. Auf ALB warten (~3-5 Min)
8. Health Check testen
9. Browser öffnen

**Dauer:** ~5-8 Minuten

---

### Option B: Manuelles Deployment (Step-by-Step)

#### Schritt 1: Account ID ersetzen

```powershell
$ACCOUNT_ID = (aws sts get-caller-identity --query Account --output text)

# Backend ServiceAccount
(Get-Content k8s\backend\serviceaccount.yaml) -replace 'ACCOUNT_ID', $ACCOUNT_ID | Set-Content k8s\backend\serviceaccount.yaml

# Backend Deployment
(Get-Content k8s\backend\deployment.yaml) -replace 'ACCOUNT_ID', $ACCOUNT_ID | Set-Content k8s\backend\deployment.yaml

# Frontend Deployment
(Get-Content k8s\frontend\deployment.yaml) -replace 'ACCOUNT_ID', $ACCOUNT_ID | Set-Content k8s\frontend\deployment.yaml
```

#### Schritt 2: Namespace erstellen

```powershell
kubectl apply -f k8s\namespace.yaml
```

#### Schritt 3: Backend deployen

```powershell
kubectl apply -f k8s\backend\serviceaccount.yaml
kubectl apply -f k8s\backend\deployment.yaml
kubectl apply -f k8s\backend\service.yaml
```

#### Schritt 4: Frontend deployen

```powershell
kubectl apply -f k8s\frontend\deployment.yaml
kubectl apply -f k8s\frontend\service.yaml
```

#### Schritt 5: ALB Ingress deployen

```powershell
kubectl apply -f k8s\ingress.yaml
```

#### Schritt 6: Pods Status prüfen

```powershell
kubectl get pods -n overcookied -w
```

**Warte bis alle Pods `Running` und `Ready 1/1` sind.**

Drücke `Ctrl+C` zum Beenden.

#### Schritt 7: ALB Ingress Status prüfen

```powershell
kubectl get ingress -n overcookied -w
```

**Warte bis `ADDRESS` Spalte ALB DNS zeigt** (~3-5 Minuten).

Drücke `Ctrl+C` zum Beenden.

---

## 🌐 Phase 5: Access Application

### Schritt 1: ALB URL holen

```powershell
$ALB_URL = (kubectl get ingress overcookied-ingress -n overcookied -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
Write-Host "Application URL: http://$ALB_URL"
```

### Schritt 2: Backend Health Check

```powershell
curl "http://$ALB_URL/health"
```

**Erwartete Ausgabe:**
```json
{"message":"Backend is running","status":"healthy"}
```

### Schritt 3: Browser öffnen

```powershell
Start-Process "http://$ALB_URL"
```

**Oder manuell:** Öffne `http://<ALB_URL>/` im Browser

---

## 🎉 Deployment Complete!

Deine Overcookied Application läuft jetzt auf:
- **EKS Cluster:** `overcookied-eks`
- **Region:** `eu-central-1`
- **URL:** `http://<ALB_URL>/`

### Monitoring & Logs

**Pod Logs ansehen:**
```powershell
# Backend Logs
kubectl logs -n overcookied -l app=overcookied-backend --tail=100 -f

# Frontend Logs
kubectl logs -n overcookied -l app=overcookied-frontend --tail=100 -f
```

**Pod Status:**
```powershell
kubectl get pods -n overcookied
kubectl describe pod <pod-name> -n overcookied
```

**Ingress Details:**
```powershell
kubectl describe ingress overcookied-ingress -n overcookied
```

---

## 💰 Optional: HPA (Horizontal Pod Autoscaler)

Für automatisches Scaling basierend auf CPU/Memory:

```powershell
kubectl apply -f k8s\backend\hpa.yaml
kubectl get hpa -n overcookied
```

---

# 🗑️ DESTROY: Cluster herunterfahren (Kosten sparen)

> [!CAUTION]
> Dies zerstört den EKS Cluster, aber **behält VPC und ECR** (Base Layer).
> Alle Application-Daten in DynamoDB bleiben erhalten.

## Option A: Automatisches Destroy (Empfohlen)

```powershell
cd C:\Users\mauri\Local Docs\overcookied\scripts
.\destroy-eks.ps1
```

**Das Script führt aus:**
1. Ingress löschen (ALB cleanup)
2. 2 Minuten warten (ENI cleanup)
3. Alle Pods/Services löschen
4. Namespace löschen
5. Terraform destroy EKS Layer
6. Base Layer verifizieren (bleibt intakt)

---

## Option B: Manuelles Destroy (Step-by-Step)

### Schritt 1: Ingress zuerst löschen (wichtig!)

```powershell
kubectl delete ingress overcookied-ingress -n overcookied
```

> [!WARNING]
> **Warte 2 Minuten** für ALB cleanup, sonst bleiben ENIs hängen!

```powershell
Start-Sleep -Seconds 120
```

### Schritt 2: Alle Kubernetes Ressourcen löschen

```powershell
kubectl delete all --all -n overcookied
kubectl delete namespace overcookied
```

### Schritt 3: EKS Terraform destroy

```powershell
cd ..\infra\eks
terraform destroy
```

Typ `yes` zur Bestätigung.

**Dauer:** ~10-15 Minuten

### Schritt 4: Base Layer verifizieren

```powershell
cd ..\base
terraform state list
```

**Erwartete Ressourcen (sollten noch da sein):**
- `aws_vpc.main`
- `aws_subnet.public[0]`, `[1]`, `[2]`
- `aws_ecr_repository.backend`
- `aws_ecr_repository.frontend`

---

## ♻️ RECREATE: Cluster wieder hochfahren

Nach einem Destroy kannst du EKS schnell wieder neu erstellen:

```powershell
cd infra\eks
terraform apply
aws eks update-kubeconfig --region eu-central-1 --name overcookied-eks
cd ..\..
.\scripts\deploy-app.ps1
```

**Dauer:** ~20-25 Minuten (schneller als initial, da VPC/ECR/Images existieren)

**Kosten:** ~€0 während zerstört, ~€156/Monat während aktiv

---

## 📊 Kosten-Übersicht

### EKS Running (24/7)
- EKS Control Plane: €73/Monat
- 2x t3.medium Nodes: €60/Monat
- ALB: €20/Monat
- DynamoDB: ~€2/Monat
- **Total: ~€155/Monat**

### Development Pattern (nur bei Bedarf)
- EKS 8h/Tag × 20 Tage = ~€30/Monat
- Base (VPC + ECR immer an): ~€1/Monat
- **Total: ~€31/Monat** 💰

---

## 🆘 Troubleshooting

### ALB erstellt sich nicht

```powershell
kubectl logs -n kube-system deployment/aws-load-balancer-controller
```

### Pods starten nicht

```powershell
kubectl describe pod <pod-name> -n overcookied
kubectl logs <pod-name> -n overcookied
```

### Backend kann nicht auf DynamoDB zugreifen

```powershell
# IRSA Annotation prüfen
kubectl get sa backend-sa -n overcookied -o yaml

# Backend Logs prüfen
kubectl logs -n overcookied -l app=overcookied-backend | Select-String "DynamoDB"
```

### Images können nicht gepullt werden

```powershell
# Prüfe ob Images in ECR sind
aws ecr list-images --repository-name overcookied-backend --region eu-central-1
aws ecr list-images --repository-name overcookied-frontend --region eu-central-1
```

---

## 📝 Notes

- **Region:** Alles läuft in `eu-central-1` (Frankfurt)
- **DynamoDB:** Existiert außerhalb von Terraform, wird nicht zerstört
- **Base Layer:** Persistent (VPC, ECR)
- **EKS Layer:** Ephemeral (kann jederzeit zerstört/neu erstellt werden)
