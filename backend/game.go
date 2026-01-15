package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/mauricedolibois/overcookied/backend/db"
)

// Message Types
const (
	MsgTypeJoinQueue     = "JOIN_QUEUE"
	MsgTypeGameStart     = "GAME_START"
	MsgTypeClick         = "CLICK"
	MsgTypeUpdate        = "UPDATE"
	MsgTypeCookieSpawn   = "COOKIE_SPAWN"
	MsgTypeCookieClick   = "COOKIE_CLICK"
	MsgTypeOpponentClick = "OPPONENT_CLICK" // New message type for red +1
	MsgTypeGameOver      = "GAME_OVER"
	MsgTypeQuit          = "QUIT_GAME"
)

type GameMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type ClickPayload struct {
	Count int `json:"count"` // 1 or 2 (for double click)
}

type GameState struct {
	TimeRemaining int    `json:"timeRemaining"`
	P1Score       int    `json:"p1Score"`
	P2Score       int    `json:"p2Score"`
	P1Name        string `json:"p1Name"`
	P2Name        string `json:"p2Name"`
}

type GameRoom struct {
	ID        string
	Player1   *Client
	Player2   *Client
	State     GameState
	Broadcast chan []byte
	Close     chan bool

	// Game Logic
	GoldenCookieActive bool
	GoldenCookieX      float64
	GoldenCookieY      float64
	DoubleClickActive  map[string]time.Time // UserID -> Expiry
	mutex              sync.Mutex
}

type GameManager struct {
	clients     map[*Client]bool
	clientsByID map[string]*Client // UserID -> Client mapping for Redis notifications
	broadcast   chan []byte
	register    chan *Client
	unregister  chan *Client
	waiting     *Client // Simple queue for 1v1 (in-memory fallback)
	clientRooms map[*Client]*GameRoom
	mutex       sync.Mutex
}

func NewGameManager() *GameManager {
	return &GameManager{
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		clientsByID: make(map[string]*Client),
		clientRooms: make(map[*Client]*GameRoom),
		waiting:     nil,
	}
}

func (gm *GameManager) Run() {
	for {
		select {
		case client := <-gm.register:
			gm.mutex.Lock()
			gm.clients[client] = true
			gm.clientsByID[client.userID] = client
			gm.mutex.Unlock()
			log.Printf("New client connected: %s", client.userID)
		case client := <-gm.unregister:
			gm.mutex.Lock()
			if _, ok := gm.clients[client]; ok {
				// Remove from Redis queue if using distributed matchmaking
				if IsRedisAvailable() {
					RemoveFromQueue(client.userID)
				}

				// Handle game disconnect if needs be
				if room, ok := gm.clientRooms[client]; ok {
					// Notify Valid Opponent
					var opponent *Client
					if room.Player1 == client {
						opponent = room.Player2
					} else {
						opponent = room.Player1
					}

					// Send Game Over (Opponent Disconnected)
					msg := GameMessage{
						Type:    MsgTypeGameOver,
						Payload: map[string]string{"winner": opponent.userID, "reason": "opponent_disconnected"},
					}
					bytes, _ := json.Marshal(msg)

					// Try to send to opponent
					select {
					case opponent.send <- bytes:
					default:
						// Opponent might be blocked or dc'ed too
					}

					// Close Room non-blocking
					go func() {
						select {
						case room.Close <- true:
						default:
						}
					}()

					delete(gm.clientRooms, room.Player1)
					delete(gm.clientRooms, room.Player2)
				}
				delete(gm.clients, client)
				delete(gm.clientsByID, client.userID)
				close(client.send)
				if gm.waiting == client {
					gm.waiting = nil
				}
				log.Printf("Client disconnected: %s", client.userID)
			}
			gm.mutex.Unlock()
		case message := <-gm.broadcast:
			gm.mutex.Lock()
			for client := range gm.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(gm.clients, client)
				}
			}
			gm.mutex.Unlock()
		}
	}
}

func (gm *GameManager) handleMessage(client *Client, msg []byte) {
	var genericMsg GameMessage
	if err := json.Unmarshal(msg, &genericMsg); err != nil {
		log.Printf("Invalid message format: %v", err)
		return
	}

	switch genericMsg.Type {
	case MsgTypeJoinQueue:
		gm.handleJoinQueue(client)
	case MsgTypeClick, MsgTypeCookieClick, MsgTypeQuit:
		// Use distributed game handling if Redis is available
		if IsRedisAvailable() {
			gm.handleDistributedGameMessage(client, genericMsg)
		} else if room, ok := gm.clientRooms[client]; ok {
			room.HandleGameMessage(client, genericMsg)
		}
	}
}

// handleDistributedGameMessage handles game messages via Redis
func (gm *GameManager) handleDistributedGameMessage(client *Client, msg GameMessage) {
	gm.mutex.Lock()
	room, ok := gm.clientRooms[client]
	gm.mutex.Unlock()

	if !ok || room == nil {
		log.Printf("No room found for client %s", client.userID)
		return
	}

	roomID := room.ID

	switch msg.Type {
	case MsgTypeClick:
		// Get current state to check for double-click powerup
		state, err := GetGameState(roomID)
		if err != nil {
			log.Printf("Failed to get game state: %v", err)
			return
		}

		points := 1
		if expiry, ok := state.DoubleClickExpiry[client.userID]; ok && time.Now().Unix() < expiry {
			points = 2
		}

		// Atomically update score in Redis
		_, err = AtomicScoreIncrement(roomID, client.userID, points)
		if err != nil {
			log.Printf("Failed to increment score: %v", err)
			return
		}

		// Publish click event to all pods
		event := GameEvent{
			RoomID:    roomID,
			EventType: EventClick,
			PlayerID:  client.userID,
			Data:      map[string]interface{}{"points": float64(points)},
		}
		PublishGameEvent(event)

	case MsgTypeCookieClick:
		// Try to atomically claim the golden cookie
		claimed, err := AtomicClaimGoldenCookie(roomID, client.userID)
		if err != nil {
			log.Printf("Failed to claim golden cookie: %v", err)
			return
		}

		if claimed {
			state, _ := GetGameState(roomID)
			event := GameEvent{
				RoomID:    roomID,
				EventType: EventGoldenClaim,
				PlayerID:  client.userID,
				Data: map[string]interface{}{
					"claimedBy": client.userID,
					"p1Score":   float64(state.P1Score),
					"p2Score":   float64(state.P2Score),
				},
			}
			PublishGameEvent(event)
		}

	case MsgTypeQuit:
		log.Printf("Processing QUIT_GAME from user: %s in room %s", client.userID, roomID)

		state, err := GetGameState(roomID)
		if err != nil {
			return
		}

		// Determine winner (the other player)
		winnerID := state.Player1ID
		if client.userID == state.Player1ID {
			winnerID = state.Player2ID
		}

		state.GameEnded = true
		state.WinnerID = winnerID
		SaveGameState(state)

		// Publish quit event
		event := GameEvent{
			RoomID:    roomID,
			EventType: EventPlayerQuit,
			PlayerID:  client.userID,
			Data:      map[string]interface{}{"winner": winnerID, "reason": "quit"},
		}
		PublishGameEvent(event)

		// Clean up
		go func() {
			time.Sleep(5 * time.Second)
			DeleteGameState(roomID)
		}()
	}
}

func (gm *GameManager) handleJoinQueue(client *Client) {
	log.Printf("Client %s joined queue", client.userID)

	// Use Redis for distributed matchmaking if available
	if IsRedisAvailable() {
		err := AddToQueue(client)
		if err != nil {
			log.Printf("Failed to add to Redis queue: %v, using in-memory fallback", err)
			// Fall through to in-memory matchmaking
		} else {
			return // Redis will handle matchmaking via RunMatchmakingLoop
		}
	}

	// In-memory fallback for single-pod mode
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	if gm.waiting != nil && gm.waiting != client {
		// Found a match!
		opponent := gm.waiting
		gm.waiting = nil
		gm.StartGame(opponent, client)
	} else {
		gm.waiting = client
	}
}

// RunMatchmakingLoop continuously checks Redis for matchmaking opportunities
func (gm *GameManager) RunMatchmakingLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		player1, player2, err := TryMatchmaking()
		if err != nil {
			log.Printf("Matchmaking error: %v", err)
			continue
		}

		if player1 != nil && player2 != nil {
			// Found a match! Create room and notify both pods
			roomID := fmt.Sprintf("%s_%s_%d", player1.UserID, player2.UserID, time.Now().Unix())

			// Create distributed game state in Redis
			if err := CreateDistributedGame(roomID, player1, player2); err != nil {
				log.Printf("Failed to create distributed game: %v", err)
				continue
			}

			match := MatchNotification{
				Player1ID: player1.UserID,
				Player2ID: player2.UserID,
				RoomID:    roomID,
				HostPodID: GetPodID(),
			}

			// Publish match notification to all pods
			if err := PublishMatchNotification(match); err != nil {
				log.Printf("Failed to publish match notification: %v", err)
			}
		}
	}
}

// SubscribeToMatchNotifications listens for match notifications from Redis Pub/Sub
func (gm *GameManager) SubscribeToMatchNotifications() {
	SubscribeToMatches(func(match MatchNotification) {
		gm.handleMatchNotification(match)
	})
}

// handleMatchNotification handles a match found by any pod
func (gm *GameManager) handleMatchNotification(match MatchNotification) {
	log.Printf("Received match notification: %s vs %s (host: %s)", match.Player1ID, match.Player2ID, match.HostPodID)

	gm.mutex.Lock()
	p1, hasP1 := gm.clientsByID[match.Player1ID]
	p2, hasP2 := gm.clientsByID[match.Player2ID]
	gm.mutex.Unlock()

	// With distributed games, we don't need both players on the same pod
	// Each pod notifies its local player about the game start

	if hasP1 {
		log.Printf("Notifying local player %s about game start", match.Player1ID)
		gm.sendGameStart(p1, match.Player2ID, "p1", match.RoomID)

		// Track which room this client is in
		gm.mutex.Lock()
		gm.clientRooms[p1] = &GameRoom{ID: match.RoomID}
		gm.mutex.Unlock()
	}

	if hasP2 {
		log.Printf("Notifying local player %s about game start", match.Player2ID)
		gm.sendGameStart(p2, match.Player1ID, "p2", match.RoomID)

		// Track which room this client is in
		gm.mutex.Lock()
		gm.clientRooms[p2] = &GameRoom{ID: match.RoomID}
		gm.mutex.Unlock()
	}

	// Only the timer pod runs the game loop
	if match.HostPodID == GetPodID() {
		log.Printf("This pod is the timer pod for room %s", match.RoomID)
		go gm.runDistributedGameLoop(match.RoomID)
	}
}

// sendGameStart sends the game start message to a player
func (gm *GameManager) sendGameStart(client *Client, opponentID, role, roomID string) {
	startMsg := GameMessage{
		Type: MsgTypeGameStart,
		Payload: map[string]interface{}{
			"opponent": opponentID,
			"role":     role,
			"roomId":   roomID,
		},
	}
	bytes, _ := json.Marshal(startMsg)
	select {
	case client.send <- bytes:
	default:
	}
}

// runDistributedGameLoop runs the game timer and broadcasts state updates via Redis
func (gm *GameManager) runDistributedGameLoop(roomID string) {
	// Wait for countdown
	time.Sleep(5 * time.Second)

	// Mark game as started
	state, err := GetGameState(roomID)
	if err != nil {
		log.Printf("Failed to get game state for %s: %v", roomID, err)
		return
	}
	state.GameStarted = true
	SaveGameState(state)

	// Broadcast initial state
	gm.broadcastGameState(roomID)

	ticker := time.NewTicker(1 * time.Second)
	gcTimer := time.NewTimer(time.Duration(5+rand.Intn(6)) * time.Second)

	defer func() {
		ticker.Stop()
		gcTimer.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			state, err := GetGameState(roomID)
			if err != nil {
				log.Printf("Game %s state not found, stopping loop", roomID)
				return
			}

			if state.GameEnded {
				return
			}

			state.TimeRemaining--
			SaveGameState(state)

			if state.TimeRemaining <= 0 {
				gm.endDistributedGame(roomID)
				return
			}

			// Broadcast state update via Redis
			gm.broadcastGameState(roomID)

		case <-gcTimer.C:
			gm.spawnDistributedGoldenCookie(roomID)
			gcTimer.Reset(time.Duration(5+rand.Intn(6)) * time.Second)
		}
	}
}

// broadcastGameState sends game state to all pods via Redis Pub/Sub
func (gm *GameManager) broadcastGameState(roomID string) {
	state, err := GetGameState(roomID)
	if err != nil {
		return
	}

	event := GameEvent{
		RoomID:    roomID,
		EventType: EventStateUpdate,
		Data: map[string]interface{}{
			"timeRemaining": state.TimeRemaining,
			"p1Score":       state.P1Score,
			"p2Score":       state.P2Score,
			"p1Name":        state.Player1Name,
			"p2Name":        state.Player2Name,
		},
	}
	PublishGameEvent(event)
}

// spawnDistributedGoldenCookie spawns a golden cookie and notifies all pods
func (gm *GameManager) spawnDistributedGoldenCookie(roomID string) {
	state, err := GetGameState(roomID)
	if err != nil {
		return
	}

	state.GoldenCookieActive = true
	state.GoldenCookieX = rand.Float64()*90 + 5
	state.GoldenCookieY = rand.Float64()*90 + 5
	SaveGameState(state)

	event := GameEvent{
		RoomID:    roomID,
		EventType: EventGoldenSpawn,
		Data: map[string]interface{}{
			"x": state.GoldenCookieX,
			"y": state.GoldenCookieY,
		},
	}
	PublishGameEvent(event)
}

// endDistributedGame ends the game and notifies all pods
func (gm *GameManager) endDistributedGame(roomID string) {
	state, err := GetGameState(roomID)
	if err != nil {
		return
	}

	state.GameEnded = true

	// Determine winner
	if state.P1Score > state.P2Score {
		state.WinnerID = state.Player1ID
	} else if state.P2Score > state.P1Score {
		state.WinnerID = state.Player2ID
	} else {
		state.WinnerID = "draw"
	}

	SaveGameState(state)

	event := GameEvent{
		RoomID:    roomID,
		EventType: EventGameEnd,
		Data: map[string]interface{}{
			"winner":  state.WinnerID,
			"p1Score": state.P1Score,
			"p2Score": state.P2Score,
		},
	}
	PublishGameEvent(event)

	// Persist game stats (only timer pod does this)
	go gm.persistGameStats(state)

	// Clean up game state after a delay
	go func() {
		time.Sleep(30 * time.Second)
		DeleteGameState(roomID)
	}()
}

// persistGameStats saves game results to DynamoDB
func (gm *GameManager) persistGameStats(state *DistributedGameState) {
	timestamp := time.Now().Unix()
	p1Won := state.P1Score > state.P2Score

	// P1
	db.SaveGame(db.CookieGame{
		GameID: state.RoomID, PlayerID: state.Player1ID, Timestamp: timestamp,
		Score: state.P1Score, OpponentScore: state.P2Score,
		Reason: "normal", Won: p1Won, WinnerID: state.WinnerID, Opponent: state.Player2ID,
		PlayerName: state.Player1Name, PlayerPicture: state.Player1Picture,
		OpponentName: state.Player2Name, OpponentPicture: state.Player2Picture,
	})
	db.UpdateUserStats(state.Player1ID, state.P1Score)

	// P2
	db.SaveGame(db.CookieGame{
		GameID: state.RoomID, PlayerID: state.Player2ID, Timestamp: timestamp,
		Score: state.P2Score, OpponentScore: state.P1Score,
		Reason: "normal", Won: !p1Won, WinnerID: state.WinnerID, Opponent: state.Player1ID,
		PlayerName: state.Player2Name, PlayerPicture: state.Player2Picture,
		OpponentName: state.Player1Name, OpponentPicture: state.Player1Picture,
	})
	db.UpdateUserStats(state.Player2ID, state.P2Score)
}

// SubscribeToGameEvents listens for game events from all pods
func (gm *GameManager) SubscribeToGameEvents() {
	SubscribeToGameEvents(func(event GameEvent) {
		gm.handleGameEvent(event)
	})
}

// handleGameEvent processes game events and sends them to local clients
func (gm *GameManager) handleGameEvent(event GameEvent) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	// Find local clients for this game
	var localClients []*Client
	for client, room := range gm.clientRooms {
		if room != nil && room.ID == event.RoomID {
			localClients = append(localClients, client)
		}
	}

	if len(localClients) == 0 {
		return // No local players for this game
	}

	var msg GameMessage

	switch event.EventType {
	case EventStateUpdate:
		msg = GameMessage{
			Type: MsgTypeUpdate,
			Payload: GameState{
				TimeRemaining: int(event.Data["timeRemaining"].(float64)),
				P1Score:       int(event.Data["p1Score"].(float64)),
				P2Score:       int(event.Data["p2Score"].(float64)),
				P1Name:        event.Data["p1Name"].(string),
				P2Name:        event.Data["p2Name"].(string),
			},
		}

	case EventGoldenSpawn:
		msg = GameMessage{
			Type:    MsgTypeCookieSpawn,
			Payload: map[string]float64{"x": event.Data["x"].(float64), "y": event.Data["y"].(float64)},
		}

	case EventGoldenClaim:
		msg = GameMessage{
			Type: MsgTypeUpdate,
			Payload: map[string]interface{}{
				"goldenCookieClaimedBy": event.Data["claimedBy"],
				"p1Score":               event.Data["p1Score"],
				"p2Score":               event.Data["p2Score"],
			},
		}

	case EventClick:
		// Send opponent click notification only to the opponent
		clickerID := event.PlayerID
		points := int(event.Data["points"].(float64))

		for _, client := range localClients {
			if client.userID != clickerID {
				oppMsg := GameMessage{
					Type:    MsgTypeOpponentClick,
					Payload: map[string]int{"count": points},
				}
				bytes, _ := json.Marshal(oppMsg)
				select {
				case client.send <- bytes:
				default:
				}
			}
		}
		return // Don't send the generic message

	case EventGameEnd:
		msg = GameMessage{
			Type:    MsgTypeGameOver,
			Payload: map[string]interface{}{"winner": event.Data["winner"]},
		}

		// Clean up client rooms
		for _, client := range localClients {
			delete(gm.clientRooms, client)
		}

	case EventPlayerQuit:
		winnerID := event.Data["winner"].(string)
		msg = GameMessage{
			Type:    MsgTypeGameOver,
			Payload: map[string]string{"winner": winnerID, "reason": "quit"},
		}

		// Clean up client rooms
		for _, client := range localClients {
			delete(gm.clientRooms, client)
		}

	default:
		return
	}

	// Send to all local clients
	bytes, _ := json.Marshal(msg)
	for _, client := range localClients {
		select {
		case client.send <- bytes:
		default:
		}
	}
}

func (gm *GameManager) StartGame(p1, p2 *Client) {
	log.Printf("Starting game between %s and %s", p1.userID, p2.userID)
	room := &GameRoom{
		ID:      fmt.Sprintf("%s_%s_%d", p1.userID, p2.userID, time.Now().Unix()),
		Player1: p1,
		Player2: p2,
		State: GameState{
			TimeRemaining: 60,        // 1 minute
			P1Name:        p1.userID, // Replace with real name later
			P2Name:        p2.userID,
		},
		Broadcast:         make(chan []byte),
		Close:             make(chan bool, 1),
		DoubleClickActive: make(map[string]time.Time),
	}

	// Notify players
	p1Start := GameMessage{Type: MsgTypeGameStart, Payload: map[string]interface{}{"opponent": p2.userID, "role": "p1"}}
	p2Start := GameMessage{Type: MsgTypeGameStart, Payload: map[string]interface{}{"opponent": p1.userID, "role": "p2"}}

	p1Bytes, _ := json.Marshal(p1Start)
	p2Bytes, _ := json.Marshal(p2Start)

	p1.send <- p1Bytes
	p2.send <- p2Bytes

	gm.clientRooms[p1] = room
	gm.clientRooms[p2] = room

	// Start Game Loop
	go room.Run()
}

// NOTE: We need to link Client to Room to route CLICK events.
// I will add a map to GameManager for this purpose.

func (room *GameRoom) Run() {
	// Broadcast initial state so clients know game is starting (and see initial time)
	room.broadcastState()

	// Wait for countdown (5 seconds)
	time.Sleep(5 * time.Second)

	ticker := time.NewTicker(1 * time.Second)
	// Golden cookie ticker (random interval 5-10s)
	gcTimer := time.NewTimer(time.Duration(5+rand.Intn(6)) * time.Second)

	defer func() {
		ticker.Stop()
		gcTimer.Stop()
	}()

	for {
		select {
		case <-room.Close:
			return
		case <-ticker.C:
			room.mutex.Lock()
			room.State.TimeRemaining--
			timeUp := room.State.TimeRemaining <= 0
			room.mutex.Unlock()

			if timeUp {
				room.EndGame()
				return
			}
			room.broadcastState()
		case <-gcTimer.C:
			room.SpawnGoldenCookie()
			gcTimer.Reset(time.Duration(5+rand.Intn(6)) * time.Second)
		}
	}
}

func (room *GameRoom) broadcastState() {
	room.mutex.Lock()
	defer room.mutex.Unlock()

	msg := GameMessage{Type: MsgTypeUpdate, Payload: room.State}
	bytes, _ := json.Marshal(msg)

	// Non-blocking send to avoid hanging if client is stuck
	select {
	case room.Player1.send <- bytes:
	default:
	}
	select {
	case room.Player2.send <- bytes:
	default:
	}
}

func (room *GameRoom) SpawnGoldenCookie() {
	room.mutex.Lock()
	room.GoldenCookieActive = true
	// Random position (0-100%)
	room.GoldenCookieX = rand.Float64()*90 + 5
	room.GoldenCookieY = rand.Float64()*90 + 5
	room.mutex.Unlock()

	msg := GameMessage{
		Type:    MsgTypeCookieSpawn,
		Payload: map[string]float64{"x": room.GoldenCookieX, "y": room.GoldenCookieY},
	}
	bytes, _ := json.Marshal(msg)
	room.Player1.send <- bytes
	room.Player2.send <- bytes
}

func (room *GameRoom) EndGame() {
	// Determine Winner
	p1Won := room.State.P1Score > room.State.P2Score
	var winnerID string
	if p1Won {
		winnerID = room.Player1.userID
	} else if room.State.P2Score > room.State.P1Score {
		winnerID = room.Player2.userID
	} else {
		winnerID = "draw"
	}

	msg := GameMessage{Type: MsgTypeGameOver,
		Payload: map[string]string{"winner": winnerID},
	}
	bytes, _ := json.Marshal(msg)
	room.Player1.send <- bytes
	room.Player2.send <- bytes

	// PERSIST GAME & UPDATE STATS
	go func() {
		timestamp := time.Now().Unix()

		// P1
		db.SaveGame(db.CookieGame{
			GameID: room.ID, PlayerID: room.Player1.userID, Timestamp: timestamp,
			Score: room.State.P1Score, OpponentScore: room.State.P2Score,
			Reason: "normal", Won: p1Won, WinnerID: winnerID, Opponent: room.Player2.userID,
			PlayerName: room.Player1.name, PlayerPicture: room.Player1.picture,
			OpponentName: room.Player2.name, OpponentPicture: room.Player2.picture,
		})
		db.UpdateUserStats(room.Player1.userID, room.State.P1Score)

		// P2
		db.SaveGame(db.CookieGame{
			GameID: room.ID, PlayerID: room.Player2.userID, Timestamp: timestamp,
			Score: room.State.P2Score, OpponentScore: room.State.P1Score,
			Reason: "normal", Won: !p1Won, WinnerID: winnerID, Opponent: room.Player1.userID,
			PlayerName: room.Player2.name, PlayerPicture: room.Player2.picture,
			OpponentName: room.Player1.name, OpponentPicture: room.Player1.picture,
		})
		db.UpdateUserStats(room.Player2.userID, room.State.P2Score)
	}()

	room.Close <- true
}

func (room *GameRoom) HandleGameMessage(client *Client, msg GameMessage) {
	room.mutex.Lock()
	defer room.mutex.Unlock()

	switch msg.Type {
	case MsgTypeClick:
		// Regular click
		points := 1
		// Check double click powerup
		if expiry, ok := room.DoubleClickActive[client.userID]; ok && time.Now().Before(expiry) {
			points = 2
		}

		if client == room.Player1 {
			room.State.P1Score += points
		} else {
			room.State.P2Score += points
		}

		// Notify opponent immediately for red "particle"
		opponent := room.Player1
		if client == room.Player1 {
			opponent = room.Player2
		}

		oppMsg := GameMessage{
			Type:    MsgTypeOpponentClick,
			Payload: map[string]int{"count": points},
		}
		oppBytes, _ := json.Marshal(oppMsg)
		select {
		case opponent.send <- oppBytes:
		default:
		}

	case MsgTypeCookieClick:
		// Attempt to claim golden cookie
		if room.GoldenCookieActive {
			room.GoldenCookieActive = false
			// Award powerup (3 second double-click bonus)
			room.DoubleClickActive[client.userID] = time.Now().Add(3 * time.Second)

			// Notify players who got it
			// Send message about who got the double click
			powerupMsg := GameMessage{
				Type: MsgTypeUpdate, // Can reuse update or new type
				Payload: map[string]interface{}{
					"goldenCookieClaimedBy": client.userID,
					"p1Score":               room.State.P1Score,
					"p2Score":               room.State.P2Score,
				},
			}
			bytes, _ := json.Marshal(powerupMsg)
			room.Player1.send <- bytes
			room.Player2.send <- bytes
			room.Player1.send <- bytes
			room.Player2.send <- bytes
		}

	case MsgTypeQuit:
		log.Printf("Processing QUIT_GAME from user: %s", client.userID)
		// Client requested to quit
		// Treat as "Resigned" -> Opponent wins or just Game Over
		otherPlayer := room.Player1
		if client == room.Player1 {
			otherPlayer = room.Player2
		}

		// Send Game Over
		msg := GameMessage{
			Type:    MsgTypeGameOver,
			Payload: map[string]string{"winner": otherPlayer.userID, "reason": "quit"},
		}
		bytes, _ := json.Marshal(msg)

		// Non-blocking sends
		select {
		case room.Player1.send <- bytes:
		default:
		}
		select {
		case room.Player2.send <- bytes:
		default:
		}

		// DO NOT PERSIST if game is aborted/quit
		log.Printf("Game %s aborted by %s, stats NOT saved. Closing room.", room.ID, client.userID)

		room.Close <- true
	}
}
