# Overcookied 🍪

A production-ready **real-time multiplayer Cookie Clicker game** with distributed architecture, cloud-native deployment, and 1v1 competitive gameplay.

## About

Overcookied is a modern take on the classic Cookie Clicker game. Players compete in real-time 1v1 matches to bake the most cookies in 60 seconds. The system features:
- **Real-time synchronization** via WebSockets (distributed across pods)
- **Distributed matchmaking** using Redis (ElastiCache/Valkey)
- **Secure authentication** with Google OAuth 2.0 + JWT tokens
- **Persistent leaderboards** backed by AWS DynamoDB
- **Horizontal scaling** with Kubernetes HPA auto-scaling
- **Production-ready** deployment on AWS EKS

## 🚀 Quick Start Guide

### Local Development (2 minutes with mocks)

```bash
# Terminal 1: Backend
cd backend
go run .

# Terminal 2: Frontend
cd frontend
npm install
npm run dev
```

Visit `http://localhost:3000` → Login with Google → Play!

### AWS EKS Deployment (45-60 minutes)

```powershell
# 1. Bootstrap AWS resources (one-time)
.\scripts\bootstrap-state.ps1
.\scripts\create-oauth-secret.ps1

# 2. Deploy infrastructure
cd infra\base && terraform apply
cd ..\eks && terraform apply

# 3. Build and deploy application
.\scripts\build-and-push.ps1
kubectl apply -f k8s\
```

## 🛠️ Tech Stack

| Layer | Technologies |
|-------|--------------|
| **Frontend** | Next.js 16.0.3 • React 19 • TypeScript 5 • Tailwind CSS 4 |
| **Backend** | Go 1.24.9 • Gorilla WebSocket 1.5.3 • JWT v5.2.1 |
| **Database** | AWS DynamoDB (serverless) |
| **Caching & State** | AWS ElastiCache (Valkey 8.0) • Redis Pub/Sub |
| **Authentication** | Google OAuth 2.0 • HS256 JWT (24h expiration) |
| **Infrastructure** | Terraform 1.9+ • Kubernetes 1.30+ • EKS • ECR |
| **Container Runtime** | Docker • AWS EKS Managed Nodes |

## 📐 Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────┐
│           AWS Application Load Balancer              │
│                  (ALB Ingress)                       │
└────────────┬─────────────────────────────┬──────────┘
             │                             │
      ┌──────▼──────┐             ┌────────▼────────┐
      │  Frontend   │             │    Backend      │
      │ (Next.js)   │             │   (Go + WS)     │
      │ Port 3000   │             │  Port 8080      │
      └─────────────┘             └────────┬────────┘
                                           │
                                  ┌────────▼────────┐
                                  │  IRSA IAM Role  │
                                  └────────┬────────┘
                                           │
                    ┌──────────────────────┼──────────────────────┐
                    │                      │                      │
            ┌───────▼────────┐   ┌────────▼─────────┐  ┌─────────▼──────┐
            │   DynamoDB     │   │  ElastiCache     │  │  Secrets Mgr   │
            │   Tables       │   │  (Valkey 8.0)    │  │  (OAuth creds) │
            │ • CookieUsers  │   │ • Matchmaking    │  └────────────────┘
            │ • CookieGames  │   │ • Pub/Sub events │
            └────────────────┘   └──────────────────┘
```

### Key Data Flows

**Authentication**:
1. User → Google OAuth login
2. Backend → Issue JWT token
3. Client → Store in localStorage
4. WebSocket → Authenticate with JWT query parameter

**Game Session**:
1. Players → Join matchmaking queue (Redis Sorted Set)
2. Matchmaking Loop → Detect 2 players, create GameRoom
3. Pub/Sub → Notify both pods of match start
4. WebSocket → Real-time score/time synchronization
5. DynamoDB → Persist game result and leaderboard

**Distributed State**:
- Game state stored in Redis (survives pod restarts)
- LocalStoreage in frontend (optimistic UI updates)
- Backend reconciliation every 1 second
- DynamoDB final persistence after match ends

## 📁 Project Structure

```
overcookied/
├── frontend/           # Next.js 16 application
│   ├── app/            # Pages & components
│   ├── hooks/          # useGameSocket (WebSocket logic)
│   └── lib/            # Auth utilities
├── backend/            # Go API + WebSocket server
│   ├── main.go         # HTTP routes & entry point
│   ├── game.go         # Game engine & room management
│   ├── websocket.go    # WebSocket pump model
│   ├── auth.go         # OAuth + JWT
│   ├── redis.go        # Matchmaking & distributed state
│   └── db/             # DynamoDB integration
├── infra/              # Terraform IaC
│   ├── base/           # VPC, ECR (persistent)
│   └── eks/            # EKS, ElastiCache, ALB (ephemeral)
├── k8s/                # Kubernetes manifests
│   ├── backend/        # Deployment, Service, HPA
│   └── frontend/       # Deployment, Service
├── docs/               # Comprehensive documentation
└── scripts/            # Deployment automation
```

## 📝 License

MIT

## 🤝 Contributing

Pull requests welcome! Ensure changes pass:
- `go test ./...` (backend)
- `npm run lint` (frontend)
- `terraform validate` (infrastructure)
