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
	broadcast   chan []byte
	register    chan *Client
	unregister  chan *Client
	waiting     *Client // Simple queue for 1v1
	clientRooms map[*Client]*GameRoom
}

func NewGameManager() *GameManager {
	return &GameManager{
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		clientRooms: make(map[*Client]*GameRoom),
		waiting:     nil,
	}
}

func (gm *GameManager) Run() {
	for {
		select {
		case client := <-gm.register:
			gm.clients[client] = true
			log.Printf("New client connected: %s", client.userID)
		case client := <-gm.unregister:
			if _, ok := gm.clients[client]; ok {
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
				close(client.send)
				if gm.waiting == client {
					gm.waiting = nil
				}
				log.Printf("Client disconnected: %s", client.userID)
			}
		case message := <-gm.broadcast:
			for client := range gm.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(gm.clients, client)
				}
			}
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
		if room, ok := gm.clientRooms[client]; ok {
			room.HandleGameMessage(client, genericMsg)
		}
	}
}

func (gm *GameManager) handleJoinQueue(client *Client) {
	log.Printf("Client %s joined queue", client.userID)
	if gm.waiting != nil && gm.waiting != client {
		// Found a match!
		opponent := gm.waiting
		gm.waiting = nil
		gm.StartGame(opponent, client)
	} else {
		gm.waiting = client
	}
}

func (gm *GameManager) StartGame(p1, p2 *Client) {
	log.Printf("Starting game between %s and %s", p1.userID, p2.userID)
	room := &GameRoom{
		ID:      fmt.Sprintf("%s_%s_%d", p1.userID, p2.userID, time.Now().Unix()),
		Player1: p1,
		Player2: p2,
		State: GameState{
			TimeRemaining: 10,        // 10 seconds (for testing)
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
	// Golden cookie ticker (random interval 20-40s)
	gcTimer := time.NewTimer(time.Duration(20+rand.Intn(21)) * time.Second)

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
			gcTimer.Reset(time.Duration(20+rand.Intn(21)) * time.Second)
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
			// Award powerup
			room.DoubleClickActive[client.userID] = time.Now().Add(5 * time.Second)

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
