# Overcookied WebSocket Architecture

This document details the real-time communication architecture used in Overcookied, built with Go (`gorilla/websocket` v1.5.3) and React 19 (Next.js 16).

## 1. Connection Lifecycle

### 1.1 Handshake & Upgrade
1.  **Client Request**: The frontend connects to `ws://DOMAIN/ws?token=JWT_TOKEN`.
2.  **Handler**: The request hits `serveWs` in `backend/websocket.go`.
3.  **JWT Validation**: The token is extracted and validated before upgrade.
4.  **Upgrade**: The standard HTTP connection is upgraded to a persistent TCP WebSocket connection using `upgrader.Upgrade()`.
5.  **Client Initialization**: A `Client` struct is created to represent this connection.
    ```go
    type Client struct {
        manager *GameManager
        conn    *websocket.Conn
        send    chan []byte  // Buffered channel for outgoing messages
        userID  string
        name    string
        picture string
    }
    ```

### 1.2 The Pump Model
To handle concurrent reads and writes safely, we use the Go concurrency pattern known as the "Pump Model". Each client connection spawns **two dedicated goroutines**:

#### A. Read Pump (`readPump`)
*   **Responsibility**: Reads *incoming* messages from the browser.
*   **Mechanism**: Runs a loop blocking on `conn.ReadMessage()`.
*   **Routing**: When a message arrives (e.g., `{"type": "CLICK"}`), it passes it to the central `GameManager` via `handleMessage()`.
*   **Cleanup**: On read error (disconnect), it unregisters the client and closes the socket.
*   **Timeouts**: Enforces read deadlines and pong timeouts (60 seconds).

#### B. Write Pump (`writePump`)
*   **Responsibility**: Push *outgoing* messages to the browser.
*   **Mechanism**: Runs a `select` loop listening on the `client.send` channel.
*   **Motivation**: Ensures only *one* goroutine ever writes to the socket, preventing race conditions. Other parts of the app (Game Loop, Timers) simply push data to the `send` channel without worrying about the socket state.
*   **Heartbeat**: Also manages a `Ticker` to send `Ping` control frames every ~54 seconds to keep the connection alive through load balancers/proxies.
*   **Write Timeout**: Enforces 10-second write deadlines for outgoing messages.

## 2. Message Routing Architecture

### 2.1 The Game Manager (`GameManager`)
The Manager is the central hub.
*   It maintains a registry of all active connections.
*   **Queues**: It handles the `MsgTypeJoinQueue`. When two players are waiting, it pairs them up.
*   **Routing**: It maps `User IDs` to `Game Rooms`. When a click message comes in, it looks up which room that player is in and forwards the message to that `GameRoom` instance.

### 2.2 The Game Room (`GameRoom`)
Each match is an isolated `GameRoom` struct running its own goroutine (`Run`).
*   **State Authority**: It holds the `GameState` (scores, time).
*   **The Loop**: A `time.Ticker` creates the game heartbeat (1 second ticks).
*   **Broadcasts**:
    *   The room does **not** write to sockets directly.
    *   It serializes the state to JSON.
    *   It attempts to push the bytes to `player.send`.
    *   **Non-Blocking Send**: It uses a `select` with a `default` case. If a client's write buffer is full (slow connection), the server drops the packet rather than blocking the entire game loop. This ensures one laggy player doesn't freeze the game for the other.

## 3. Data Flow Example: "Cookie Click"

1.  **User Action**: Player clicks cookie. Frontend `useGameSocket` sends JSON: `{"type": "CLICK"}`.
2.  **Network**: Message travels over WS to Backend.
3.  **Backend Read**: `Client.readPump` receives message -> `GameManager`.
4.  **Logic**: `GameManager` finds `GameRoom`. `GameRoom` increments score.
5.  **Broadcast**: `GameRoom` broadcasts new state update.
6.  **Backend Write**: JSON payload pushed to `Client.send`. `writePump` wakes up, writes to TCP socket.
7.  **Frontend Update**: Browser receives `UPDATE` message. React updates state.

## 4. Key Security & Performance Features
*   **Concurrency Safety**: All shared state is protected. The `GameRoom` uses `sync.Mutex` during state updates to ensure the Ticker (writes) and ReadPump (reads/writes) don't corrupt memory.
*   **Keep-Alive**: Heartbeat (Ping/Pong) ensures dead connections are detected and cleaned up.
*   **Data Integrity**: JSON marshaling ensures structured data vs raw byte streams.

## 5. Protocol Reference

### Client -> Server
*   `JOIN_QUEUE`: Request to enter the matchmaking pool.
*   `CLICK`: Player clicked the cookie (standard +1).
*   `COOKIE_CLICK`: Player clicked the Golden Cookie.
*   `QUIT_GAME`: Player requests to leave/surrender the game.

### Server -> Client
*   `GAME_START`: Match found, game is beginning (starts countdown).
*   `UPDATE`: Periodic state sync (Scores, Timer, Powerups).
*   `OPPONENT_CLICK`: Notification that opponent clicked (used for visual particles).
*   `COOKIE_SPAWN`: Golden Cookie appeared at coordinates (x,y).
*   `GAME_OVER`: Game finished (Win/Loss/Draw/Quit). Payload contains winner and reason.
