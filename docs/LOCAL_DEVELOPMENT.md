# 🛠️ Local Development Environment

This guide describes how to develop Overcookied locally without deploying to AWS.

## Quick Start (with Mocks) - 2 Minutes

The simplest method for local development is **Mock Mode**, which replaces all AWS services with in-memory implementations.

### 1. Start Backend

```bash
cd backend
cp .env.example .env  # if it doesn't exist, create one
# Ensure USE_MOCKS=true (it's the default)
go run .
```

Backend runs on `http://localhost:8080`

### 2. Start Frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend runs on `http://localhost:3000`

### 3. Test

Open `http://localhost:3000` and log in with Google.

---

## Mock Mode vs Production Mode

### Mock Mode (Recommended for Local Development)

Set in `backend/.env`:
```env
USE_MOCKS=true
```

All AWS services are replaced:

| Service | Mock Behavior |
|---------|---------------|
| **DynamoDB** | In-memory storage with sample data (3 users, 3 games) |
| **ElastiCache (Redis/Valkey)** | In-memory queue for matchmaking |
| **AWS Secrets Manager** | Loads OAuth credentials from `.env` |
| **OAuth** | Still uses real Google OAuth (requires credentials) |

**Benefits:**
- ✅ No AWS account needed (except for Google OAuth)
- ✅ Fast startup (~1 second)
- ✅ Deterministic test data
- ✅ No latency from AWS services
- ✅ Works offline (except OAuth)

### Production Mode (Full AWS)

Set in `backend/.env`:
```env
USE_MOCKS=false
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret
AWS_REGION=eu-central-1
REDIS_ENDPOINT=localhost:6379  # or ElastiCache endpoint
DYNAMODB_TABLE_USERS=CookieUsers
DYNAMODB_TABLE_GAMES=CookieGames
GOOGLE_OAUTH_SECRET_NAME=overcookied/google-oauth
```

---

## System Requirements

- **Go**: 1.24.9 or higher
---

## Google OAuth Configuration

### 1. Get OAuth Credentials

1. Go to [Google Cloud Console - Credentials](https://console.cloud.google.com/apis/credentials)
2. Select or create an OAuth 2.0 Client ID (Web application)
3. Add authorized URIs:

**Authorized JavaScript Origins:**
```
http://localhost:3000
http://localhost:8080
```

**Authorized Redirect URIs:**
```
http://localhost:8080/auth/google/callback
```

4. Copy **Client ID** and **Client Secret**

### 2. Backend Setup

```bash
cd backend
# Create .env (copy template if it exists)
cat > .env << 'EOF'
# Google OAuth Credentials
GOOGLE_CLIENT_ID=your-client-id-here
GOOGLE_CLIENT_SECRET=your-client-secret-here

# Local OAuth callback
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

# JWT Secret (any random string)
JWT_SECRET=local-jwt-secret-change-in-production

# Frontend URL (no trailing slash)
FRONTEND_URL=http://localhost:3000

# Port
PORT=8080

# Mock Mode (true for local dev without AWS)
USE_MOCKS=true
EOF

# Run backend
go run .
```

Expected output:
```
Connected to OAuth provider
[MOCK] In-memory DynamoDB initialized
[MOCK] In-memory Redis initialized for matchmaking
Server starting on port 8080
```

### 3. Frontend Setup

```bash
cd frontend
npm install

# Create .env.local
echo "NEXT_PUBLIC_API_URL=http://localhost:8080" > .env.local

# Start dev server
npm run dev
```

Visit `http://localhost:3000`

---

## Testing Flows

### Login Flow
1. Visit `http://localhost:3000`
2. Click "Login with Google"
3. Authenticate with your Google account
4. Redirected to dashboard with JWT token stored

### Matchmaking
1. Open two browser windows (or incognito)
2. Log in with different Google accounts
3. Click "Find Match" in both
4. Game starts automatically with 5-second countdown

### Leaderboard (Mock Data)
- Alice Baker: 1,500 points
- Bob Chef: 1,200 points
- Charlie Cook: 900 points

---

## Troubleshooting

| Error | Solution |
|-------|----------|
| `redirect_uri_mismatch` | OAuth callback URL in .env must exactly match Google Console |
| CORS errors | Check `FRONTEND_URL` env var (must be `http://localhost:3000` without trailing slash) |
| "Failed to fetch" | Verify backend is running on port 8080 and `NEXT_PUBLIC_API_URL` is correct |
| WebSocket errors | Usually caused by CORS - verify frontend and backend URLs match |
| Mock mode not working | Ensure `USE_MOCKS=true` in `.env` |

---

## Useful Commands

```bash
# Backend: Run tests
cd backend && go test ./...

# Backend: Format code
go fmt ./...

# Frontend: Run linter
cd frontend && npm run lint

# Frontend: Build for production
npm run build

# Check Go version
go version

# Check Node version
node --version
```

---

## Architecture Diagram (Local)

```
Frontend (React 19)          Backend (Go 1.24)
├─ page.tsx                 ├─ main.go (HTTP server)
├─ game/                    ├─ game.go (game logic)
├─ hooks/                   ├─ websocket.go (WS hub)
│  └─ useGameSocket.ts      ├─ auth.go (JWT + OAuth)
└─ lib/auth.ts              ├─ db/dynamo.go (DynamoDB)
     │                      └─ redis.go (matchmaking)
     │ HTTP/WebSocket         │
     └──────────────────────┘
         (localhost:3000→8080)
                │
     ┌──────────┴──────────┐
     │                     │
  DynamoDB            ElastiCache
  (Mock)              (Mock)
cd backend && go build .

# Backend Tests
cd backend && go test ./...

# Frontend Build (Produktion)
cd frontend && npm run build

# Frontend Linting
cd frontend && npm run lint
```
