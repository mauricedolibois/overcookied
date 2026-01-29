# Overcookied 🍪

Ein Echtzeit-Multiplayer Cookie Clicker Spiel mit verteilter Architektur auf AWS EKS.

## Was ist Overcookied?

Zwei Spieler treten gegeneinander an, um in 60 Sekunden die meisten Cookies zu backen. Klick schnell, fang goldene Cookies (+5 Punkte) und klettere in der Rangliste!

**Features:**
- Echtzeit 1v1-Matches via WebSockets
- Google OAuth Login mit JWT Sessions
- Verteiltes Matchmaking über mehrere Pods (Redis/Valkey)
- Persistente Bestenlisten (DynamoDB)
- Auto-Scaling Kubernetes Deployment

## Schnellstart

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

```
                    ┌─────────────────────┐
                    │   AWS ALB Ingress   │
                    └─────────┬───────────┘
                              │
            ┌─────────────────┼─────────────────┐
            │                 │                 │
     ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐
     │  Frontend   │   │  Backend    │   │  Backend    │
     │  (Next.js)  │   │  Pod 1      │   │  Pod N      │
     └─────────────┘   └──────┬──────┘   └──────┬──────┘
                              │                 │
                    ┌─────────┴─────────────────┴─────────┐
                    │                                     │
             ┌──────▼──────┐                      ┌───────▼──────┐
             │ ElastiCache │                      │   DynamoDB   │
             │  (Valkey)   │                      │              │
             └─────────────┘                      └──────────────┘
```

## Projektstruktur

```
overcookied/
├── backend/          # Go API + WebSocket Server
├── frontend/         # Next.js Anwendung
├── infra/            # Terraform (base + eks)
├── k8s/              # Kubernetes Manifeste
├── scripts/          # Deployment Skripte
└── docs/             # Dokumentation
```

## Dokumentation

| Dokument | Beschreibung |
|----------|--------------|
| [Local Development](docs/LOCAL_DEVELOPMENT.md) | Setup-Anleitung für lokale Entwicklung |
| [Deployment](docs/DEPLOYMENT.md) | AWS EKS Deployment Schritte |
| [Architecture](docs/architecture/ARCHITECTURE.md) | System-Design Details |
| [Testing](docs/TESTING.md) | Test-Strategie und Befehle |
| [Runbook](docs/RUNBOOK.md) | Betrieb und Fehlerbehebung |

## AWS Deployment

```powershell
# 1. Setup (einmalig)
.\scripts\bootstrap-state.ps1
.\scripts\create-oauth-secret.ps1

# 2. Infrastruktur
cd infra\base && terraform apply
cd ..\eks && terraform apply

# 3. Deployment
.\scripts\build-and-push.ps1
.\scripts\deploy-app.ps1
```

## Lizenz

MIT
