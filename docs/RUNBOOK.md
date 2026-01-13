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

### Option A: Automatisches Deployment

#### Schritt 1: Application deployen

```powershell
cd C:\Users\mauri\Local Docs\overcookied
.\scripts\deploy-app.ps1
```

**Das Script führt aus:**
1. Account ID in Manifests ersetzen
2. Namespace erstellen
3. JWT Secret erstellen (aus AWS Secrets Manager falls vorhanden)
4. Backend deployen (ServiceAccount, Deployment, Service)
5. Frontend deployen
6. ALB Ingress erstellen
7. Auf Pods warten
8. Auf ALB warten (~3-5 Min)
9. Health Check testen
10. Browser öffnen

**Dauer:** ~5-8 Minuten

#### Schritt 2: Route 53 DNS und OAuth für Custom Domain konfigurieren

```powershell
.\scripts\update-oauth-config.ps1 -UpdateDNS -RestartBackend
```

**Das Script führt aus:**
1. ALB hostname ermitteln
2. Route 53 A-Record für `overcookied.de` → neuer ALB
3. OAuth ConfigMap mit `https://overcookied.de` URLs updaten
4. Backend Pods neu starten

#### Schritt 3: Deployment verifizieren

```powershell
# DNS prüfen (via Google DNS bei lokalen Problemen)
nslookup overcookied.de 8.8.8.8

# Health Check
curl https://overcookied.de/health --resolve overcookied.de:443:$(nslookup overcookied.de 8.8.8.8 | Select-String "Address" | Select-Object -Last 1).ToString().Split(" ")[-1]
```

#### Schritt 4: Im Browser öffnen

```powershell
Start-Process "https://overcookied.de"
```

**Checkliste:**
- [ ] HTTPS funktioniert (🔒 Schloss-Symbol)
- [ ] Login mit Google funktioniert
- [ ] Redirect nach Login auf Dashboard

> [!NOTE]
> Bei DNS-Problemen: `ipconfig /flushdns` oder DNS auf 8.8.8.8 ändern.

---

### Option B: Manuelles Deployment (Step-by-Step)

<details>
<summary>📖 Manuelle Schritte anzeigen (für Debugging)</summary>

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

#### Schritt 8: Route 53 und OAuth konfigurieren

Siehe [Option A Schritt 2](#schritt-2-route-53-dns-und-oauth-für-custom-domain-konfigurieren).

</details>

---

## 🌐 Phase 5: Custom Domain & HTTPS Setup (Referenz)

> [!NOTE]
> Wenn du **Option A** verwendet hast, ist Phase 5 bereits in Schritt 2 enthalten!
> Dieser Abschnitt dient als Referenz für Details und Troubleshooting.

> [!CAUTION]
> **Dieser Schritt ist PFLICHT für produktiven Betrieb!**
> Ohne diesen Schritt funktioniert nur HTTP über die ALB-URL, nicht HTTPS über overcookied.de.

### Schritt 1: OAuth URLs aktualisieren und Route 53 DNS updaten

Nach dem ALB provisioning, update DNS und OAuth ConfigMap:

```powershell
cd C:\Users\mauri\Local Docs\overcookied
.\scripts\update-oauth-config.ps1 -UpdateDNS -RestartBackend
```

**Das Script führt aus:**
1. Wartet auf ALB hostname
2. Updated Route 53 A-Record für `overcookied.de` → zeigt auf neuen ALB
3. Updated OAuth ConfigMap mit HTTPS URLs (`https://overcookied.de`)
4. Startet Backend neu für neue Config

> [!NOTE]
> Bei jedem neuen ALB (z.B. nach Cluster-Recreate) ändert sich die ALB-URL!
> Route 53 muss dann erneut aktualisiert werden.

### Schritt 2: SSL Zertifikat verifizieren

```powershell
aws acm describe-certificate --certificate-arn arn:aws:acm:eu-central-1:032073356456:certificate/75eb55b7-dde0-4aac-9836-278ed5d8063c --query "Certificate.Status"
```

**Erwartete Ausgabe:** `"ISSUED"`

### Schritt 3: DNS Resolution testen

```powershell
# Via öffentlichen DNS (empfohlen bei lokalen DNS-Problemen)
nslookup overcookied.de 8.8.8.8
```

> [!TIP]
> Falls lokale DNS-Auflösung nicht funktioniert (z.B. bei Fritz.box Router), 
> liegt das oft an DNS-Caching. Lösung: `ipconfig /flushdns` oder direkt 8.8.8.8 als DNS verwenden.

### Schritt 4: Application testen

```powershell
Start-Process "https://overcookied.de"
```

**Teste:**
- [ ] HTTPS funktioniert (Schloss-Symbol)
- [ ] Login mit Google funktioniert
- [ ] Redirect nach Login korrekt

---

## 🎉 Deployment Complete!

Deine Overcookied Application läuft jetzt auf:
- **EKS Cluster:** `overcookied-eks`
- **Region:** `eu-central-1`
- **URL:** `https://overcookied.de` 🔒

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
# 1. EKS Cluster erstellen (~15-20 Min)
cd infra\eks
terraform apply

# 2. kubectl konfigurieren
aws eks update-kubeconfig --region eu-central-1 --name overcookied-eks

# 3. Application deployen (erstellt JWT Secret automatisch)
cd ..\..
.\scripts\deploy-app.ps1

# 4. Custom Domain & OAuth konfigurieren (updated Route 53 mit neuem ALB)
.\scripts\update-oauth-config.ps1 -UpdateDNS -RestartBackend
```

**Dauer:** ~20-25 Minuten (schneller als initial, da VPC/ECR/Images existieren)

### Was bleibt erhalten nach Destroy?
- ✅ Route 53 Hosted Zone & Nameserver
- ✅ ACM Zertifikat (bereits validiert)
- ✅ DynamoDB Tabellen
- ✅ AWS Secrets Manager (OAuth Credentials)
- ✅ ECR Images
- ✅ Google OAuth Console Konfiguration

### Was wird neu erstellt?
- 🔄 EKS Cluster
- 🔄 ALB (neuer Hostname → Route 53 muss updated werden)
- 🔄 JWT Secret (automatisch durch deploy-app.ps1)
- 🔄 Kubernetes ConfigMaps

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

### OAuth Login fehlschlägt (invalid_state)

Dies bedeutet Cookie-Probleme zwischen Pods. Prüfe:
1. HTTPS ist korrekt konfiguriert
2. Alle Backend Pods nutzen denselben JWT Secret

```powershell
# JWT Secret prüfen
kubectl get secret jwt-secret -n overcookied

# Wenn nicht vorhanden, neu erstellen:
$secret = -join ((65..90) + (97..122) + (48..57) | Get-Random -Count 32 | ForEach-Object {[char]$_})
kubectl create secret generic jwt-secret --from-literal=JWT_SECRET=$secret -n overcookied
kubectl rollout restart deployment/overcookied-backend -n overcookied
```

### OAuth Redirect falsch (localhost statt Domain)

Frontend verwendet noch alte URLs. Rebuild und Deploy:

```powershell
.\scripts\build-and-push.ps1
kubectl rollout restart deployment/overcookied-frontend -n overcookied
```

### Route 53 DNS zeigt auf alten ALB

```powershell
.\scripts\update-oauth-config.ps1 -UpdateDNS -RestartBackend
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
