# Overcookied - Architekturdiagramm

## System-Übersicht

```mermaid
graph TB
    subgraph Client["Frontend (Next.js)"]
        Browser[Browser]
        Pages[Pages/Routes]
        Hooks[Custom Hooks]
        Components[UI Components]
        
        Browser --> Pages
        Pages --> Hooks
        Pages --> Components
        Hooks --> Components
    end

    subgraph Backend["Backend (Go)"]
        HTTPServer[HTTP Server]
        WSHandler[WebSocket Handler]
        GameManager[Game Manager]
        GameRoom[Game Room]
        Auth[Auth Service]
        
        HTTPServer --> WSHandler
        HTTPServer --> Auth
        WSHandler --> GameManager
        GameManager --> GameRoom
    end

    subgraph Database["AWS DynamoDB"]
        UsersTable[(Users Table)]
        GamesTable[(Games Table)]
    end

    subgraph External["External Services"]
        GoogleOAuth[Google OAuth 2.0]
    end

    Browser -->|HTTP/HTTPS| HTTPServer
    Browser -->|WebSocket| WSHandler
    Auth -->|JWT Verify| HTTPServer
    Auth -->|OAuth Flow| GoogleOAuth
    GameRoom -->|Store Stats| UsersTable
    GameRoom -->|Store History| GamesTable
    Auth -->|CRUD| UsersTable
```

## Authentifizierungsfluss

```mermaid
sequenceDiagram
    actor User
    participant Frontend
    participant Backend
    participant Google
    participant DynamoDB

    User->>Frontend: Klick "Login with Google"
    Frontend->>Backend: GET /auth/google
    Backend->>Google: OAuth Redirect
    Google->>User: Google Login Page
    User->>Google: Credentials
    Google->>Backend: Callback mit Code
    Backend->>Google: Code Exchange
    Google->>Backend: User Profile
    Backend->>DynamoDB: Create/Update User
    DynamoDB->>Backend: User Record
    Backend->>Backend: Generate JWT
    Backend->>Frontend: Redirect mit JWT
    Frontend->>Frontend: Store JWT in localStorage
    Frontend->>User: Redirect to Dashboard
```

## WebSocket-Architektur

```mermaid
graph TB
    subgraph "Client Connection"
        Client[Client Browser]
        WS[WebSocket Connection]
    end

    subgraph "Backend WebSocket Layer"
        ServeWS[serveWs Handler]
        ClientStruct[Client Struct]
        ReadPump[Read Pump Goroutine]
        WritePump[Write Pump Goroutine]
        SendChannel[send channel]
    end

    subgraph "Game Logic Layer"
        Manager[Game Manager]
        Queue[Match Queue]
        Room1[Game Room 1]
        Room2[Game Room 2]
        RoomN[Game Room N]
    end

    Client -->|ws://server/ws| ServeWS
    ServeWS -->|HTTP Upgrade| WS
    ServeWS -->|Create| ClientStruct
    ClientStruct -->|spawn| ReadPump
    ClientStruct -->|spawn| WritePump
    ReadPump -->|Messages| Manager
    Manager -->|Matchmaking| Queue
    Queue -->|Create Pairs| Room1
    Queue -->|Create Pairs| Room2
    Queue -->|Create Pairs| RoomN
    Room1 -->|State Updates| SendChannel
    Room2 -->|State Updates| SendChannel
    RoomN -->|State Updates| SendChannel
    SendChannel -->|JSON| WritePump
    WritePump -->|TCP| WS
    WS -->|Network| Client

    style "Client Connection" fill:#e3f2fd
    style "Backend WebSocket Layer" fill:#fff3e0
    style "Game Logic Layer" fill:#f3e5f5
```

## Game Room Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Waiting: Players Join Queue
    Waiting --> Matched: 2 Players Found
    Matched --> Starting: Create GameRoom
    Starting --> Countdown: Broadcast GAME_START
    Countdown --> Playing: 5 Second Countdown
    
    Playing --> Playing: Process Clicks
    Playing --> Playing: Spawn Golden Cookie
    Playing --> Playing: Broadcast UPDATE
    Playing --> GameOver: Time Expires
    Playing --> GameOver: Player Quits
    
    GameOver --> SaveStats: Determine Winner
    SaveStats --> Cleanup: Write to DynamoDB
    Cleanup --> [*]

    note right of Playing
        Game Loop (1 sec tick)
        - Update Timer
        - Process Events
        - Broadcast State
    end note

    note right of SaveStats
        - Update User Stats
        - Store Game History
        - Calculate Rankings
    end note
```

## Nachrichtenfluss: Cookie Click

```mermaid
sequenceDiagram
    participant UI as Frontend UI
    participant Hook as useGameSocket Hook
    participant WS as WebSocket
    participant Read as ReadPump
    participant Manager as GameManager
    participant Room as GameRoom
    participant Write as WritePump

    UI->>Hook: User clicks cookie
    Hook->>Hook: Optimistic UI update
    Hook->>WS: send('{"type":"CLICK"}')
    WS->>Read: Message received
    Read->>Manager: Route message
    Manager->>Room: Forward to correct room
    
    Room->>Room: Mutex Lock
    Room->>Room: Increment score
    Room->>Room: Mutex Unlock
    
    Room->>Room: Prepare UPDATE message
    Room->>Write: Push to send channel
    Write->>WS: Write JSON to socket
    WS->>Hook: Receive UPDATE
    Hook->>Hook: Update local state
    Hook->>UI: Re-render with new score
```

## Datenmodell

```mermaid
erDiagram
    USERS ||--o{ GAMES : participates
    
    USERS {
        string UserID PK
        string GoogleID
        string Name
        string Email
        string AvatarURL
        int TotalGames
        int Wins
        int Losses
        int HighScore
        timestamp CreatedAt
        timestamp UpdatedAt
    }
    
    GAMES {
        string GameID PK
        string Player1ID FK
        string Player2ID FK
        int Player1Score
        int Player2Score
        string Winner
        string EndReason
        int Duration
        timestamp PlayedAt
    }
```

## Deployment-Architektur

```mermaid
graph TB
    subgraph "Client Layer"
        Users[Users/Browsers]
    end

    subgraph "Kubernetes Cluster"
        Ingress[Ingress Controller]
        
        subgraph "Frontend Service"
            FE1[Frontend Pod 1]
            FE2[Frontend Pod 2]
            FEService[Frontend Service]
        end
        
        subgraph "Backend Service"
            BE1[Backend Pod 1]
            BE2[Backend Pod 2]
            BEService[Backend Service]
        end
    end

    subgraph "AWS Cloud"
        DDB[(DynamoDB)]
    end

    subgraph "External"
        OAuth[Google OAuth]
    end

    Users -->|HTTPS| Ingress
    Ingress --> FEService
    Ingress --> BEService
    FEService --> FE1
    FEService --> FE2
    BEService --> BE1
    BEService --> BE2
    BE1 --> DDB
    BE2 --> DDB
    BE1 --> OAuth
    BE2 --> OAuth

    style "Client Layer" fill:#e3f2fd
    style "Kubernetes Cluster" fill:#fff3e0
    style "AWS Cloud" fill:#f3e5f5
    style "External" fill:#e8f5e9
```

## Technologie-Stack

```mermaid
graph LR
    subgraph Frontend
        NextJS[Next.js 16.0.3]
        React[React 19]
        Tailwind[Tailwind CSS]
        TS[TypeScript]
        
        NextJS --> React
        NextJS --> Tailwind
        NextJS --> TS
    end

    subgraph Backend
        Go[Go/Golang]
        Gorilla[gorilla/websocket]
        NetHTTP[net/http]
        AWSSDK[AWS SDK]
        
        Go --> Gorilla
        Go --> NetHTTP
        Go --> AWSSDK
    end

    subgraph Infrastructure
        Docker[Docker]
        K8s[Kubernetes]
        AWS[AWS DynamoDB]
        
        Docker --> K8s
        K8s --> AWS
    end

    Frontend -.->|WebSocket| Backend
    Frontend -.->|HTTP/REST| Backend
    Backend -.->|Persist| Infrastructure

    style Frontend fill:#61dafb20
    style Backend fill:#00add820
    style Infrastructure fill:#32629620
```

## Concurrency-Modell (Backend)

```mermaid
graph TB
    subgraph "Main Goroutine"
        Main[main.go]
        HTTPServer[HTTP Server]
    end

    subgraph "Per-Client Goroutines"
        RP1[ReadPump 1]
        WP1[WritePump 1]
        RP2[ReadPump 2]
        WP2[WritePump 2]
        RPN[ReadPump N]
        WPN[WritePump N]
    end

    subgraph "Per-Room Goroutines"
        Room1[GameRoom.Run 1]
        Room2[GameRoom.Run 2]
        RoomN[GameRoom.Run N]
        
        Ticker1[Ticker 1]
        Ticker2[Ticker 2]
        TickerN[Ticker N]
        
        Room1 --> Ticker1
        Room2 --> Ticker2
        RoomN --> TickerN
    end

    subgraph "Shared State Protection"
        Mutex1[sync.Mutex 1]
        Mutex2[sync.Mutex 2]
        MutexN[sync.Mutex N]
    end

    Main --> HTTPServer
    HTTPServer -->|spawn| RP1
    HTTPServer -->|spawn| WP1
    HTTPServer -->|spawn| RP2
    HTTPServer -->|spawn| WP2
    HTTPServer -->|spawn| RPN
    HTTPServer -->|spawn| WPN

    RP1 -.->|protected by| Mutex1
    Ticker1 -.->|protected by| Mutex1
    RP2 -.->|protected by| Mutex2
    Ticker2 -.->|protected by| Mutex2
    RPN -.->|protected by| MutexN
    TickerN -.->|protected by| MutexN

    Room1 --> WP1
    Room1 --> WP2
    Room2 --> WP1
    Room2 --> WP2

    style "Main Goroutine" fill:#fff3e0
    style "Per-Client Goroutines" fill:#e1f5ff
    style "Per-Room Goroutines" fill:#f3e5f5
    style "Shared State Protection" fill:#ffebee
```

## Message Types (WebSocket Protocol)

```mermaid
graph TB
    subgraph "Client → Server"
        C1[JOIN_QUEUE]
        C2[CLICK]
        C3[COOKIE_CLICK]
        C4[QUIT_GAME]
    end

    subgraph "Server → Client"
        S1[GAME_START]
        S2[UPDATE]
        S3[OPPONENT_CLICK]
        S4[COOKIE_SPAWN]
        S5[GAME_OVER]
    end

    subgraph "Game State Machine"
        State[Game State]
    end

    C1 -->|Matchmaking| State
    C2 -->|Score +1| State
    C3 -->|Special Bonus| State
    C4 -->|End Game| State
    
    State -->|Initialize| S1
    State -->|Periodic Sync| S2
    State -->|Notify| S3
    State -->|Event| S4
    State -->|Finish| S5

    style "Client → Server" fill:#e3f2fd
    style "Server → Client" fill:#fff3e0
    style "Game State Machine" fill:#f3e5f5
```
