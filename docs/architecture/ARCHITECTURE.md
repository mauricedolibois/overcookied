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

### 4.2. WebSocket & Matchmaking
-   **Connection**: Client connects to `/ws` with `?userId=...` (Validation to be improved).
-   **Queueing**: `MsgTypeJoinQueue` message adds the client to a waiting list in `GameManager`.
-   **Matching**: When 2 players are in queue, `GameManager` creates a `GameRoom`.
-   **Room**: The `GameRoom` is an isolated struct managing the state for those 2 players.

### 4.3. The Game Loop (`backend/game.go`)
The heart of the backend is the `GameRoom.Run()` goroutine:
1.  **Init**: Broadcasts `MsgTypeGameStart` and waits 5 seconds (countdown).
2.  **Loop**: A `time.Ticker` ticks every 1 second.
    -   Decrements `TimeRemaining`.
    -   Checks if `TimeRemaining <= 0` -> **EndGame**.
    -   Broadcasts `MsgTypeUpdate` (Score, Time) to both clients.
3.  **Events**:
    -   **Clicks**: Clients send `MsgTypeClick`. Backend updates score.
    -   **Golden Cookie**: Random timer spawns a golden cookie. Client claiming it sends `MsgTypeCookieClick`.
    -   **Power-ups**: Backend manages power-up state (e.g., 2x multiplier).

### 4.4. State Synchronization
-   **Authority**: Backend is the source of truth.
-   **Optimistic UI**: Frontend updates UI immediately on click (particles, counter) but reconciles with Backend `MsgTypeUpdate`.
-   **Concurrency**: `GameRoom` uses `sync.Mutex` to protect shared state (`GameState`) from concurrent writes (Ticker vs WebSocket ReadPump).

### 4.5. Data Persistence
-   **DynamoDB Tables**:
    -   `Users`: Stores profile (ID, Name, Email, Stats).
    -   `Games`: Stores match history (Scores, Winner, Timestamp).
-   **End of Game**:
    -   `GameRoom` determines winner (or draw).
    -   Asynchronously writes `Game` record and updates `User` stats in DynamoDB.

## 5. Security & Improvements (Current State)
-   **CORS**: Currently permissive for development.
-   **Auth**: JWT based, but WebSocket connection currently relies on query param `userId` (needs hardening to validate JWT on handshake).
-   **State**: In-memory game state; server restart kills active games.

## 6. Frontend Architecture
-   **Pages**:
    -   `Dashboard`: Fetches History/Leaderboard via REST API.
    -   `Game`: Connects via WebSocket.
-   **Hooks**:
    -   `useGameSocket.ts`: Encapsulates all WebSocket logic. Handles connection, message routing, and state updates. It exposes simple methods (`sendClick`, `quitGame`) to the UI.
