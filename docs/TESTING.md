# Test Documentation

This document describes the unit tests for the Overcookied project.

## Running Tests

### Backend Tests (Go)

```bash
cd backend
go test ./... -v
```

To run only the fast unit tests (skipping integration tests):
```bash
go test ./... -v -short
```

### Frontend Tests (TypeScript/Vitest)

```bash
cd frontend
npm test
```

For watch mode during development:
```bash
npm run test:watch
```

### Run All Tests (Before Deploy)

The `scripts/build-and-push.ps1` script automatically runs all tests before building Docker images.

## Backend Test Structure

Tests follow Go conventions and are located alongside the source code:

```
backend/
├── mocks/
│   ├── dynamo_mock.go           # MockDynamoDB implementation
│   ├── dynamo_mock_test.go      # MockDynamoDB tests
│   ├── redis_mock.go            # MockRedis implementation
│   └── redis_mock_test.go       # MockRedis tests
├── db/
│   ├── dynamo.go                # Real DynamoDB operations
│   └── dynamo_integration_test.go  # Integration tests (requires AWS)
```

### Mock Tests (`mocks/`)

These tests verify the in-memory mock implementations used for local development:

#### DynamoDB Mock Tests
- **User Operations**: SaveUser, GetUser, score preservation on update
- **Leaderboard**: GetTopUsers with proper sorting
- **Game History**: SaveGame, GetGamesByPlayer (sorted by timestamp)
- **Statistics**: CountGamesByPlayer, GetUserStats
- **Concurrency**: Thread-safe score updates

#### Redis Mock Tests
- **Queue Operations**: AddToQueue, RemoveFromQueue, GetQueueLength
- **Matchmaking**: TryMatch (FIFO ordering)
- **Pub/Sub**: PublishMatch, Subscribe
- **Game State**: SaveGameState, GetGameState, DeleteGameState, events

### Integration Tests (`db/`)

Integration tests only run when AWS is configured:

```bash
# Set AWS credentials first
export AWS_REGION=eu-central-1
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...

go test ./db/... -v
```

These tests are automatically **skipped** in local development when AWS is not configured.

## Frontend Test Structure

```
frontend/
├── __tests__/
│   ├── setup.ts              # Test environment setup (localStorage mock)
│   ├── auth.test.ts          # Auth service tests
│   ├── websocket.test.ts     # WebSocket URL generation tests
│   └── dataMapping.test.ts   # API-to-Component data mapping tests
├── vitest.config.ts          # Vitest configuration
```

### Auth Tests (`auth.test.ts`)

**Critical security tests for JWT handling:**
- Session management (getCurrentUser, saveUser, removeUser)
- JWT token expiration validation
- Malformed token handling
- Authentication state checks

### WebSocket Tests (`websocket.test.ts`)

**Connectivity tests for the game server connection:**
- Development mode URL generation (localhost:8080)
- Production mode URL derivation from window.location
- Secure WebSocket (wss://) for HTTPS origins
- Port handling

### API Data Mapping Tests (`dataMapping.test.ts`)

**Contract tests between backend API and frontend components:**
- Leaderboard API response transformation
- Game history API response transformation
- Field name mapping (e.g., `opponent` → `opponentId`)
- Optional field handling

## Test Philosophy

1. **Business Logic First**: Tests focus on game logic, matchmaking, and data integrity
2. **No Styling Tests**: UI/styling tests are excluded (getMedalEmoji, colors, etc.)
3. **API Contracts**: Frontend tests verify correct handling of backend responses
4. **Security Critical**: JWT validation is thoroughly tested
5. **Integration Optional**: AWS integration tests run only when configured
