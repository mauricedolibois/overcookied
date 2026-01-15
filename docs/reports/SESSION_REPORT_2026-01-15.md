# 📋 Session Report: Distributed Gaming mit Redis/Valkey

**Datum:** 15. Januar 2026  
**Projekt:** Overcookied - Idle Game auf AWS EKS

---

## 🎯 Zusammenfassung

In dieser Session haben wir das Matchmaking-System von single-pod auf eine **vollständig verteilte Architektur** umgestellt. Spieler können jetzt auf verschiedenen Kubernetes Pods gegeneinander spielen, indem AWS ElastiCache (Valkey) für State-Synchronisation und Pub/Sub-Kommunikation verwendet wird.

---

## ✅ Durchgeführte Arbeiten

### 1. EKS Cluster Neuaufbau
- Terraform State Lock gelöst (force-unlock)
- Orphaned IAM Roles/Policies manuell gelöscht
- EKS Cluster `overcookied-eks` neu erstellt
- AWS Load Balancer Controller via Helm deployed

### 2. ElastiCache Valkey Setup
- **Problem:** `aws_elasticache_cluster` unterstützt Valkey nicht
- **Lösung:** Migration zu `aws_elasticache_replication_group` API
- Valkey Cluster manuell erstellt und in Terraform importiert
- Security Groups für EKS-Nodes → Valkey konfiguriert

### 3. Distributed Matchmaking Implementation
- Redis-basierte Matchmaking Queue (Sorted Set)
- Distributed Lock für Race-Condition-freies Matching
- Pub/Sub für Match-Notifications an alle Pods

### 4. Distributed Game State
- Game State in Redis Hash gespeichert
- Atomische Score-Updates mit Redis Transactions
- Atomisches Golden Cookie Claiming (First-come-first-served)
- Game Events via Redis Pub/Sub synchronisiert

### 5. Frontend Countdown Fix
- GAME_START Payload erweitert um initialen State
- Frontend zeigt 5-Sekunden Countdown korrekt an

---

## 🐛 Probleme & Lösungen

### Problem 1: Valkey API-Inkompatibilität

**Symptom:** Terraform Fehler bei `aws_elasticache_cluster`

```
InvalidParameterValue: This API doesn't support Valkey engine. 
Please use CreateReplicationGroup API for Valkey cluster creation.
```

**Ursache:** AWS ElastiCache hat zwei APIs - `CreateCacheCluster` (alt) und `CreateReplicationGroup` (neu). Valkey wird nur von der neuen API unterstützt.

**Lösung:**
```hcl
# Vorher (funktioniert NICHT für Valkey)
resource "aws_elasticache_cluster" "valkey" { ... }

# Nachher (korrekt)
resource "aws_elasticache_replication_group" "valkey" {
  engine         = "valkey"
  engine_version = "8.0"
  ...
}
```

---

### Problem 2: Match-Hosting auf verschiedenen Pods

**Symptom:** Logs zeigten "Host pod missing players: hasP1=true, hasP2=false"

**Ursache:** Der alte Code versuchte beide Spieler auf einem Pod zu haben:
```go
// Alt: Funktioniert nur wenn beide Spieler auf demselben Pod sind
if hasP1 && hasP2 {
    gm.StartGame(p1, p2)
}
```

**Lösung:** Komplett neue Architektur mit verteiltem State:
```go
// Neu: Jeder Pod benachrichtigt seine lokalen Spieler
if hasP1 {
    gm.sendGameStart(p1, match.Player2ID, "p1", match.RoomID)
}
if hasP2 {
    gm.sendGameStart(p2, match.Player1ID, "p2", match.RoomID)
}

// Timer-Pod verwaltet Game Loop
if match.HostPodID == GetPodID() {
    go gm.runDistributedGameLoop(match.RoomID)
}
```

---

### Problem 3: Security Group Referenz

**Symptom:** Redis-Verbindung timeout nach Pod-Restart

**Ursache:** Terraform referenzierte die falsche Security Group:
```hcl
# Falsch: Node Security Group (existiert, aber Pods nutzen sie nicht direkt)
security_groups = [aws_security_group.nodes.id]

# Richtig: EKS Cluster Security Group (die Pods tatsächlich nutzen)
security_groups = [aws_eks_cluster.main.vpc_config[0].cluster_security_group_id]
```

**Lösung:** Security Group manuell aktualisiert und Terraform korrigiert.

---

### Problem 4: Race Conditions bei Scores

**Symptom:** Theoretisch konnten zwei Pods gleichzeitig den Score erhöhen und sich überschreiben.

**Lösung:** Redis WATCH/MULTI Transactions:
```go
func AtomicScoreIncrement(roomID, playerID string, points int) (*DistributedGameState, error) {
    err := redisClient.Watch(ctx, func(tx *redis.Tx) error {
        // Lese aktuellen State
        state := getState(tx, key)
        
        // Aktualisiere Score
        state.P1Score += points
        
        // Atomisch speichern
        _, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
            pipe.Set(ctx, key, newState, ttl)
            return nil
        })
        return err
    }, key)
}
```

---

### Problem 5: Countdown nicht angezeigt

**Symptom:** 5-Sekunden Countdown vor Spielbeginn wurde nicht angezeigt.

**Ursache:** GAME_START enthielt keinen initialen Game State, Frontend hatte keine Daten zum Anzeigen.

**Lösung:**
```go
// Backend: Initialen State mitsenden
startMsg := GameMessage{
    Type: MsgTypeGameStart,
    Payload: map[string]interface{}{
        "opponent":      opponentID,
        "role":          role,
        "roomId":        roomID,
        "timeRemaining": state.TimeRemaining,  // NEU
        "p1Name":        state.Player1Name,    // NEU
        "p2Name":        state.Player2Name,    // NEU
    },
}
```

```typescript
// Frontend: State aus GAME_START lesen
case 'GAME_START':
    setGameStatus('PLAYING');
    if (msg.payload.timeRemaining !== undefined) {
        setGameState({
            timeRemaining: msg.payload.timeRemaining,
            p1Name: msg.payload.p1Name,
            p2Name: msg.payload.p2Name,
            ...
        });
    }
```

---

## 📊 Neue Architektur

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              AWS Cloud                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         EKS Cluster                                  │    │
│  │                                                                      │    │
│  │   ┌──────────────┐              ┌──────────────┐                    │    │
│  │   │   Pod A      │              │   Pod B      │                    │    │
│  │   │  (Backend)   │              │  (Backend)   │                    │    │
│  │   │              │              │              │                    │    │
│  │   │  Player 1 ◄──┼──────────────┼──► Player 2  │                    │    │
│  │   │  WebSocket   │   Pub/Sub    │  WebSocket   │                    │    │
│  │   └──────┬───────┘              └──────┬───────┘                    │    │
│  │          │                             │                            │    │
│  │          │    Redis State + Events     │                            │    │
│  │          └──────────────┬──────────────┘                            │    │
│  │                         │                                           │    │
│  │                         ▼                                           │    │
│  │              ┌─────────────────────┐                                │    │
│  │              │   ElastiCache       │                                │    │
│  │              │   (Valkey 8.0)      │                                │    │
│  │              │                     │                                │    │
│  │              │  • Matchmaking Queue│                                │    │
│  │              │  • Game States      │                                │    │
│  │              │  • Pub/Sub Events   │                                │    │
│  │              │  • Distributed Locks│                                │    │
│  │              └─────────────────────┘                                │    │
│  │                                                                      │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 🔄 Datenfluss

### Matchmaking Flow

```
1. Player 1 → Pod A: JOIN_QUEUE
2. Pod A → Redis: ZADD matchmaking:queue {player1}

3. Player 2 → Pod B: JOIN_QUEUE  
4. Pod B → Redis: ZADD matchmaking:queue {player2}

5. Pod A (Matchmaking Loop):
   - SETNX matchmaking:lock (distributed lock)
   - ZRANGE matchmaking:queue 0 1 (get 2 players)
   - ZREM matchmaking:queue (remove matched)
   - SET game:{id} (create game state)
   - PUBLISH match:notify (notify all pods)

6. All Pods receive match notification:
   - Check if local player is involved
   - Send GAME_START to local WebSocket client
```

### Game Event Flow

```
Click Event:
  Player → Pod → Redis (atomic increment) → Pub/Sub → All Pods → WebSocket clients

Golden Cookie:
  Timer Pod → Redis (SETNX for claim) → Pub/Sub → All Pods → WebSocket clients

Timer:
  Timer Pod → Redis (decrement) → Pub/Sub → All Pods → WebSocket clients
```

---

## 📦 Redis Data Structures

| Key Pattern | Type | Beschreibung | TTL |
|-------------|------|--------------|-----|
| `overcookied:matchmaking:queue` | Sorted Set | Wartende Spieler (Score = Timestamp) | - |
| `overcookied:matchmaking:lock` | String | Distributed Lock für Matchmaking | 2s |
| `overcookied:game:{roomId}` | String (JSON) | Game State | 10m |
| `overcookied:match:notify` | Pub/Sub | Match Found Notifications | - |
| `overcookied:game:events` | Pub/Sub | Game Events (clicks, timer, etc.) | - |

---

## 🔑 Wichtige Ressourcen

| Ressource | Wert |
|-----------|------|
| Domain | overcookied.de |
| EKS Cluster | overcookied-eks |
| Region | eu-central-1 |
| Valkey Endpoint | `overcookied-valkey.aakwdp.ng.0001.euc1.cache.amazonaws.com:6379` |
| Valkey Node Type | cache.t3.micro |
| Valkey Engine | Valkey 8.0 |

---

## 📁 Geänderte Dateien

### Backend

| Datei | Änderungen |
|-------|------------|
| `backend/redis.go` | Neue Funktionen: `CreateDistributedGame()`, `SaveGameState()`, `GetGameState()`, `AtomicScoreIncrement()`, `AtomicClaimGoldenCookie()`, `PublishGameEvent()`, `SubscribeToGameEvents()` |
| `backend/game.go` | Neue Funktionen: `handleDistributedGameMessage()`, `runDistributedGameLoop()`, `broadcastGameState()`, `sendGameStart()` (erweitert), `SubscribeToGameEvents()` |
| `backend/main.go` | Game Events Subscription gestartet |

### Frontend

| Datei | Änderungen |
|-------|------------|
| `frontend/app/hooks/useGameSocket.ts` | GAME_START Handler setzt initialen Game State |

### Infrastructure

| Datei | Änderungen |
|-------|------------|
| `infra/eks/elasticache.tf` | `aws_elasticache_cluster` → `aws_elasticache_replication_group`, Security Group Fix |
| `infra/eks/outputs.tf` | Valkey Endpoint Output aktualisiert |
| `k8s/backend/deployment.yaml` | `REDIS_ENDPOINT` Environment Variable |
| `k8s/backend/redis-configmap.yaml` | Neu: Redis ConfigMap |
| `scripts/deploy-app.ps1` | Valkey Endpoint auto-detection |

---

## 📝 Lessons Learned

1. **Valkey braucht ReplicationGroup API** - `aws_elasticache_cluster` funktioniert nur für Redis/Memcached
2. **EKS Pods nutzen Cluster Security Group** - Nicht die Node Security Group
3. **Distributed State braucht atomische Operationen** - Redis WATCH/MULTI für Race Conditions
4. **Pub/Sub für Echtzeit-Sync** - Alle Pods bekommen Events sofort
5. **Timer-Pod Konzept** - Ein Pod verwaltet den Timer, alle anderen reagieren auf Events
6. **State im GAME_START** - Frontend braucht initialen State für Countdown

---

## ✨ Status

**Distributed Gaming ist vollständig implementiert:**

- ✅ Matchmaking über Redis Queue
- ✅ Spieler können auf verschiedenen Pods sein
- ✅ Atomische Score-Updates
- ✅ Atomisches Golden Cookie Claiming
- ✅ Game State Synchronisation via Pub/Sub
- ✅ 5-Sekunden Countdown funktioniert

🔗 **https://overcookied.de**
