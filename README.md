# Overcookied 🍪

Ein Echtzeit-Multiplayer Cookie Clicker Spiel mit verteilter Architektur auf AWS EKS.

## Was ist Overcookied?

In Overcookied treten zwei zufällig ausgewählte Spieler in einem 60 Sekunden Match gegeneinander an. Wer am Ende die meisten Klicks hat, gewinnt. Zusätzlich spawnt alle 5-10 Sekunden ein “Golden Cookie”. Der Spieler, der zuerst auf den Golden Cookie klickt, erhält einen kurzen „Double Click”-Bonus, bei dem jeder Klick doppelt gewertet wird. Klick' so schnell du kannst und klettere in der Rangliste!

**Features:**
- Echtzeit 1v1-Matches via WebSockets
- Google OAuth Login mit JWT Sessions
- Verteiltes Matchmaking über mehrere Pods (Redis/Valkey)
- Persistente Bestenlisten (DynamoDB)
- Auto-Scaling Kubernetes Deployment

## Quick Start Guide

### Lokale Entwicklung

```bash
# Backend (Terminal 1)
cd backend
go run .

# Frontend (Terminal 2)
cd frontend
npm install && npm run dev
```

Öffne `http://localhost:3000` → Mit Google einloggen → Spielen!

> **Hinweis:** Im lokalen Modus werden In-Memory Mocks für Redis und DynamoDB verwendet.

### Tests ausführen

```bash
# Backend
cd backend && go test ./... -v

# Frontend
cd frontend && npm test
```

## Tech Stack

| Schicht | Technologie |
|---------|-------------|
| Frontend | Next.js 16, React 19, TypeScript, Tailwind CSS |
| Backend | Go 1.24, Gorilla WebSocket, JWT |
| Datenbank | AWS DynamoDB |
| Cache | AWS ElastiCache (Valkey 8.0) |
| Auth | Google OAuth 2.0 |
| Infra | Terraform, Kubernetes, AWS EKS |

## Architektur
<img width="942" height="912" alt="AWS Architektur" src="https://github.com/user-attachments/assets/e98cdd6c-0336-43b5-8c28-92216462d4ca" />

## AWS Deployment

### Voraussetzungen

Folgende Tools müssen installiert und konfiguriert sein:

| Tool | Version | Zweck |
|------|---------|-------|
| [AWS CLI](https://aws.amazon.com/cli/) | v2+ | AWS-Ressourcen verwalten |
| [Terraform](https://www.terraform.io/) | >= 1.9 | Infrastruktur provisionieren |
| [Docker](https://www.docker.com/) | 20+ | Container-Images bauen |
| [kubectl](https://kubernetes.io/docs/tasks/tools/) | 1.31+ | Kubernetes-Cluster verwalten |
| [Go](https://go.dev/) | 1.24+ | Backend bauen & testen |
| [Node.js](https://nodejs.org/) | 22+ | Frontend bauen & testen |
| [PowerShell](https://github.com/PowerShell/PowerShell) | 7+ | Deployment-Scripte ausführen |

Außerdem wird benötigt:
- **AWS Account** mit konfiguriertem `aws configure` (Access Key, Region `eu-central-1`)
- **Google OAuth 2.0 Credentials** — erstellt in der [Google Cloud Console](https://console.cloud.google.com/apis/credentials) (OAuth 2.0 Client-ID vom Typ "Webanwendung")
- **(Optional) Domain** — für HTTPS wird eine eigene Domain (z.B. `overcookied.de`) mit Route 53 Hosted Zone benötigt

### AWS-Deployment Schritte

```powershell
# 1. Terraform State Backend einrichten (einmalig)
.\scripts\bootstrap-state.ps1

# 2. Google OAuth + JWT Secrets in AWS Secrets Manager anlegen (einmalig)
.\scripts\create-oauth-secret.ps1

# 3. Basis-Infrastruktur erstellen (VPC, ECR Repositories)
cd infra\base
terraform init
terraform apply
cd ..\..

# 4. Docker-Images bauen, testen und in ECR pushen
.\scripts\build-and-push.ps1

# 5. EKS-Cluster erstellen und Anwendung deployen
.\scripts\create-cluster.ps1
```

> **Hinweis:** `create-cluster.ps1` führt automatisch `deploy-app.ps1` und `update-oauth-config.ps1` aus.
> Nach dem Deployment müssen die angezeigten Redirect-URLs in der Google Cloud Console eingetragen werden.

### Teardown

```powershell
# EKS-Cluster zerstören (VPC & ECR bleiben erhalten)
.\scripts\destroy-eks.ps1

# Basis-Infrastruktur komplett entfernen (optional)
cd infra\base && terraform destroy
```

### Script-Übersicht

| Script | Beschreibung |
|--------|-------------|
| `bootstrap-state.ps1` | Erstellt einmalig den S3-Bucket und die DynamoDB-Tabelle für das Terraform Remote State Backend. |
| `create-oauth-secret.ps1` | Speichert die Google OAuth-Credentials und einen generierten JWT-Secret im AWS Secrets Manager. |
| `build-and-push.ps1` | Führt Backend- und Frontend-Tests aus, baut die Docker-Images und pusht sie in die ECR-Repositories. |
| `create-cluster.ps1` | Provisioniert den EKS-Cluster via Terraform, konfiguriert kubectl und deployt die gesamte Anwendung. |
| `deploy-app.ps1` | Deployt alle Kubernetes-Ressourcen (Namespace, ConfigMaps, Secrets, Deployments, Services, Ingress) in den EKS-Cluster. |
| `update-oauth-config.ps1` | Aktualisiert nach dem Deployment die OAuth-ConfigMap mit der korrekten Domain-URL und optional den Route 53 DNS-Eintrag. |
| `destroy-eks.ps1` | Räumt alle Kubernetes-Ressourcen auf und zerstört den EKS-Cluster via Terraform, lässt die Basis-Infrastruktur (VPC, ECR) bestehen. |

## Lizenz
Uniprojekt der Hochschule der Medien Stuttgart. Alle Rechte vorbehalten. Nicht für kommerzielle Zwecke verwenden.
