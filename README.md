# Overcookied

A modern multiplayer Cookie Clicker game built with Next.js 16 (frontend) and Go (backend), deployable on AWS EKS.

## 🎮 Features

- **Real-time Multiplayer**: WebSocket-based game sessions
- **Google OAuth**: Secure authentication
- **Leaderboard**: Global rankings with DynamoDB persistence
- **Game History**: Track all your matches
- **Production-Ready**: EKS deployment with Terraform IaC

## 🏗️ Architecture

### Application Stack
- **Frontend**: Next.js 16.0.3 with React 19 and Tailwind CSS
- **Backend**: Go HTTP server with WebSocket support (Gorilla)
- **Database**: AWS DynamoDB (serverless)
- **Authentication**: Google OAuth 2.0

### AWS Infrastructure (EKS)
- **Base Layer** (persistent): VPC, ECR repositories
- **EKS Layer** (ephemeral): EKS cluster, managed node groups, ALB ingress
- **Security**: IRSA (IAM Roles for Service Accounts) for DynamoDB access
- **Cost-Optimized**: Public nodes, no NAT Gateway (~€30-50/month)

## 📁 Project Structure

```
overcookied/
├── frontend/              # Next.js application
│   ├── app/               # Next.js app directory
│   ├── public/            # Static assets
│   └── Dockerfile
├── backend/               # Go backend application
│   ├── main.go            # HTTP server & WebSocket
│   ├── db/                # DynamoDB integration
│   └── Dockerfile
├── infra/                 # Terraform Infrastructure as Code
│   ├── base/              # VPC, ECR (persistent)
│   └── eks/               # EKS cluster (ephemeral)
├── k8s/                   # Kubernetes manifests
│   ├── backend/           # Backend deployment, service, HPA
│   ├── frontend/          # Frontend deployment, service
│   └── ingress.yaml       # ALB ingress configuration
├── scripts/               # Deployment automation (PowerShell)
└── DEPLOYMENT.md          # Detailed deployment guide
```

## 🚀 Quick Start (Local Development)

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Visit http://localhost:3000

### Backend

```bash
cd backend
go run .
```

Visit http://localhost:8080/health

## ☁️ AWS EKS Deployment

### Prerequisites

- AWS CLI (configured)
- Terraform >= 1.9
- kubectl >= 1.30
- Docker
- Helm >= 3.16

### Quick Deploy

```powershell
# 1. Bootstrap (one-time)
.\scripts\bootstrap-state.ps1
.\scripts\create-oauth-secret.ps1

# 2. Deploy Base Infrastructure
cd infra\base
terraform init
terraform apply

# 3. Build & Push Images
cd ..\..
.\scripts\build-and-push.ps1

# 4. Deploy EKS Cluster
cd infra\eks
terraform init
terraform apply

# 5. Deploy Application
cd ..\..
.\scripts\deploy-app.ps1
```

**Detailed instructions**: See [DEPLOYMENT.md](DEPLOYMENT.md)

## 🗑️ Destroy Infrastructure

```powershell
# Destroy only EKS (keeps VPC & ECR)
.\scripts\destroy-eks.ps1

# Later recreate EKS (fast, uses existing VPC/ECR)
cd infra\eks
terraform apply
```

## 🔒 Security Features

✅ **IRSA**: No AWS credentials in containers  
✅ **Secrets Manager**: OAuth credentials stored securely  
✅ **Security Groups**: Minimal attack surface  
✅ **Resource Limits**: Prevents resource exhaustion  
✅ **ECR Scanning**: Automated vulnerability detection  

## 📊 Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│ Application Load Balancer (ALB)                         │
│ http://<alb-dns>/                                       │
└──────────────┬──────────────────────────────────────────┘
               │
       ┌───────┴────────┐
       │                │
┌──────▼───────┐ ┌─────▼────────┐
│  Frontend    │ │   Backend    │
│  (Next.js)   │ │   (Go)       │
│  Port 3000   │ │   Port 8080  │
│              │ │   WebSocket  │
└──────────────┘ └──────┬───────┘
                        │
                ┌───────┴────────┐
                │   IRSA Role    │
                └───────┬────────┘
                        │
            ┌───────────▼──────────────┐
            │    DynamoDB Tables       │
            │  - CookieUsers           │
            │  - CookieGames           │
            └──────────────────────────┘
```

## 💰 Cost Estimate (eu-central-1)

- **EKS Control Plane**: ~€73/month
- **EC2 Nodes (2x t3.medium)**: ~€60/month
- **ALB**: ~€20/month
- **DynamoDB On-Demand**: ~€1-5/month (low traffic)
- **ECR Storage**: ~€1/month
- **NAT Gateway**: €0 (using public nodes)

**Total**: ~€155/month (can be reduced to €30-50/month by destroying EKS when not in use)

## 🛠️ Customization

### Change Node Instance Type

Edit `infra/eks/terraform.tfvars`:

```hcl
node_instance_types = ["t3.small"]  # Smaller for cost savings
node_desired_size   = 1              # Reduce replicas
```

### Enable HTTPS

1. Create ACM certificate in AWS Console
2. Uncomment TLS annotations in `k8s/ingress.yaml`
3. Update `alb.ingress.kubernetes.io/certificate-arn`

### Add Custom Domain

1. Create Route53 hosted zone
2. Update `k8s/ingress.yaml` with domain in `spec.rules[].host`
3. Create CNAME record pointing to ALB DNS

## 📝 License

MIT

## 🤝 Contributing

Pull requests welcome! Please ensure your changes pass:
- Go tests: `go test ./...`
- Terraform validation: `terraform validate`
- Kubernetes dry-run: `kubectl apply --dry-run=client`
