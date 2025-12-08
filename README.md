# Overcookied

A modern Cookie Clicker game built with Next.js 16 (frontend) and Go (backend), deployable on Kubernetes.

## Architecture

- **Frontend**: Next.js 16.0.3 with React 19 and Tailwind CSS
- **Backend**: Go HTTP server with health check and API endpoints
- **Deployment**: Kubernetes manifests for container orchestration

## Project Structure

```
cookie-clicker/
├── frontend/          # Next.js frontend application
│   ├── app/           # Next.js app directory
│   ├── public/        # Static assets
│   ├── Dockerfile     # Frontend container image
│   └── package.json   # Node.js dependencies
└── backend/           # Go backend application
    ├── main.go        # HTTP server implementation
    ├── go.mod         # Go module definition
    └── Dockerfile     # Backend container image


## Local Development

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
go build -o bin/server .
./bin/server
```

Visit http://localhost:8080/health

## Building Docker Images

### Frontend

```bash
cd frontend
docker build -t cookie-clicker-frontend:latest .
```

### Backend

```bash
cd backend
docker build -t cookie-clicker-backend:latest .
```
