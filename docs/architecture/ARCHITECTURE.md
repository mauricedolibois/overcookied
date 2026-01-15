# Overcookied Architecture Documentation

## 1. Overview
Overcookied is a real-time multiplayer competitive clicker game. Players compete 1v1 to bake the most cookies within a time limit. The system features real-time state synchronization, matchmaking, and persistent user statistics.

## 2. Tech Stack

### Frontend
- **Framework**: [Next.js](https://nextjs.org/) (React) with App Router.
- **Styling**: [Tailwind CSS](https://tailwindcss.com/) for responsive and utility-first styling.
- **Language**: TypeScript.
- **State Management**: React Hooks (`useState`, `useEffect`) + Custom Hooks (`useGameSocket`).
- **Communication**: WebSockets (native browser API).

### Backend
- **Language**: Go (Golang).
- **Web Server**: Standard `net/http`.
- **WebSockets**: [gorilla/websocket](https://github.com/gorilla/websocket).
- **Database**: AWS DynamoDB.
- **Cache/State Store**: AWS ElastiCache (Valkey 8.0) for distributed matchmaking and game state.
- **Authentication**: Google OAuth 2.0 + JWT (JSON Web Tokens).

## 3. Project Structure

```
overcookied/
├── backend/                # Go Backend
│   ├── db/                 # Database interaction layer (DynamoDB)
│   ├── main.go             # Entry point, HTTP server, Routes
│   ├── game.go             # Game logic, Game Loop, Room management
│   ├── websocket.go        # WebSocket connection handling (Hub, Client)
│   ├── auth.go             # OAuth and JWT logic
│   ├── redis.go            # ElastiCache/Valkey integration for distributed state
│   └── go.mod              # Go dependencies
│
└── frontend/               # Next.js Frontend
    ├── app/                # App Router Pages
    │   ├── page.tsx        # Landing Page
    │   ├── login/          # Login Page
    │   ├── dashboard/      # User Dashboard (History, Leaderboard)
    │   └── game/           # Main Game Arena
    ├── components/         # Reusable UI Components
    ├── lib/                # Utilities (Auth, API helpers)
    └── public/             # Static assets
```

## 4. Key Components & Workflows

### 4.1. Authentication Flow
1.  **Login**: User clicks "Login with Google" on Frontend.
2.  **Redirect**: Backend redirects to Google OAuth provider.
3.  **Callback**: Google callbacks to Backend with code.
4.  **Token Exchange**: Backend exchanges code for Google User Profile.
5.  **Session Creation**: Backend creates a user record in DynamoDB (if new) and issues a JWT.
6.  **Client Storage**: Frontend stores JWT in `localStorage`.
7.  **Verification**: Subsequent requests (API & WebSocket) validate the JWT.

### 4.2. WebSocket & Matchmaking (Distributed via ElastiCache)
-   **Connection**: Client connects to `/ws` with JWT token for authentication.
-   **Queueing**: `MsgTypeJoinQueue` message adds the client to a Redis Sorted Set (`overcookied:matchmaking:queue`) with timestamp-based ordering.
-   **Distributed Lock**: Matchmaking uses Redis `SetNX` for distributed locking to prevent race conditions across pods.
-   **Matching**: When 2 players are in queue, any pod can atomically pop them and create a match.
-   **Pub/Sub Notification**: Match notifications are broadcast via Redis Pub/Sub (`overcookied:match:notify`) to all pods.
-   **Room Assignment**: The pod containing both players' WebSocket connections hosts the `GameRoom`.
-   **Cross-Pod Coordination**: If players are on different pods, game state is synchronized via Redis.

### 4.3. The Game Loop (`backend/game.go` + `backend/redis.go`)
The heart of the backend is the `GameRoom.Run()` goroutine with distributed state support:
1.  **Init**: Creates `DistributedGameState` in Redis, broadcasts `MsgTypeGameStart` and waits 5 seconds (countdown).
2.  **Loop**: A `time.Ticker` ticks every 1 second.
    -   Decrements `TimeRemaining` (synced to Redis).
    -   Checks if `TimeRemaining <= 0` -> **EndGame**.
    -   Broadcasts `MsgTypeUpdate` (Score, Time) to both clients.
3.  **Events**:
    -   **Clicks**: Clients send `MsgTypeClick`. Backend uses `AtomicScoreIncrement()` for thread-safe Redis updates.
    -   **Golden Cookie**: Random timer spawns a golden cookie. `AtomicClaimGoldenCookie()` ensures only one player can claim it.
    -   **Power-ups**: Managed via Redis with TTL-based expiration tracking.
4.  **Distributed State Keys**:
    -   Game state: `overcookied:game:{roomId}`
    -   Events: `overcookied:game:events` (Pub/Sub channel)

### 4.4. State Synchronization (Distributed)
-   **Authority**: AWS ElastiCache (Valkey) is the distributed source of truth for game state.
-   **Optimistic UI**: Frontend updates UI immediately on click (particles, counter) but reconciles with Backend `MsgTypeUpdate`.
-   **Local Concurrency**: `GameRoom` uses `sync.Mutex` to protect in-memory state from concurrent writes.
-   **Distributed Concurrency**: Redis `WATCH`/`MULTI` transactions ensure atomic updates across pods.
-   **Event Broadcasting**: Redis Pub/Sub propagates game events to all backend pods for real-time sync.

### 4.5. Data Persistence
-   **DynamoDB Tables**:
    -   `Users`: Stores profile (ID, Name, Email, Stats).
    -   `Games`: Stores match history (Scores, Winner, Timestamp).
-   **End of Game**:
    -   `GameRoom` determines winner (or draw).
    -   Asynchronously writes `Game` record and updates `User` stats in DynamoDB.

## 5. Security & Infrastructure
-   **CORS**: Configured for production domains.
-   **Auth**: JWT based with token validation on WebSocket handshake.
-   **State**: Distributed game state via AWS ElastiCache (Valkey 8.0).
    -   Game state survives individual pod restarts.
    -   Matchmaking queue is shared across all backend pods.
    -   TTL-based cleanup (10 minutes) for orphaned game states.
-   **Horizontal Scaling**: Backend pods can scale independently; ElastiCache provides shared state coordination.

## 6. Frontend Architecture
-   **Pages**:
    -   `Dashboard`: Fetches History/Leaderboard via REST API.
    -   `Game`: Connects via WebSocket.
-   **Hooks**:
    -   `useGameSocket.ts`: Encapsulates all WebSocket logic. Handles connection, message routing, and state updates. It exposes simple methods (`sendClick`, `quitGame`) to the UI.
