package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mauricedolibois/overcookied/backend/mocks"
	"github.com/redis/go-redis/v9"
)

// QueueEntry represents a player waiting in the matchmaking queue
type QueueEntry struct {
	UserID   string `json:"userId"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	PodID    string `json:"podId"`
	JoinedAt int64  `json:"joinedAt"`
}

// MatchNotification is sent via Pub/Sub when a match is found
type MatchNotification struct {
	Player1ID string `json:"player1Id"`
	Player2ID string `json:"player2Id"`
	RoomID    string `json:"roomId"`
	HostPodID string `json:"hostPodId"`
}

var (
	redisClient  *redis.Client
	ctx          = context.Background()
	podID        string
	useMockRedis bool
)

const (
	matchmakingQueueKey = "overcookied:matchmaking:queue"
	matchmakingLockKey  = "overcookied:matchmaking:lock"
	matchNotifyChannel  = "overcookied:match:notify"
	queueTTL            = 30 * time.Second // Players auto-removed after 30s
)

// InitRedis initializes the Redis/Valkey connection
func InitRedis() error {
	useMockRedis = mocks.IsMockMode()

	if useMockRedis {
		log.Println("[REDIS] Running in MOCK MODE - using in-memory matchmaking")
		podID = mocks.GetMockRedis().GetPodID()
		return nil
	}

	redisAddr := os.Getenv("REDIS_ENDPOINT")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Default for local dev
	}

	hostname, _ := os.Hostname()
	podID = fmt.Sprintf("%s_%d", hostname, time.Now().UnixNano())

	redisClient = redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     "", // ElastiCache doesn't use password by default
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})

	// Test connection
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v. Using in-memory fallback.", err)
		return err
	}

	log.Printf("Connected to Redis/Valkey at %s (Pod: %s)", redisAddr, podID)
	return nil
}

// AddToQueue adds a player to the matchmaking queue
func AddToQueue(client *Client) error {
	if useMockRedis {
		return mocks.GetMockRedis().AddToQueue(client.userID, client.name, client.picture)
	}

	if redisClient == nil {
		return fmt.Errorf("redis not initialized")
	}

	entry := QueueEntry{
		UserID:   client.userID,
		Name:     client.name,
		Picture:  client.picture,
		PodID:    podID,
		JoinedAt: time.Now().Unix(),
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// Add to queue with score as timestamp (for ordering)
	err = redisClient.ZAdd(ctx, matchmakingQueueKey, redis.Z{
		Score:  float64(entry.JoinedAt),
		Member: string(entryJSON),
	}).Err()

	if err != nil {
		return err
	}

	log.Printf("Added %s to matchmaking queue", client.userID)
	return nil
}

// RemoveFromQueue removes a player from the matchmaking queue
func RemoveFromQueue(userID string) error {
	if useMockRedis {
		return mocks.GetMockRedis().RemoveFromQueue(userID)
	}

	if redisClient == nil {
		return fmt.Errorf("redis not initialized")
	}

	// Get all queue entries and remove matching user
	entries, err := redisClient.ZRange(ctx, matchmakingQueueKey, 0, -1).Result()
	if err != nil {
		return err
	}

	for _, entryStr := range entries {
		var entry QueueEntry
		if err := json.Unmarshal([]byte(entryStr), &entry); err != nil {
			continue
		}
		if entry.UserID == userID {
			redisClient.ZRem(ctx, matchmakingQueueKey, entryStr)
			log.Printf("Removed %s from matchmaking queue", userID)
			break
		}
	}

	return nil
}

// TryMatchmaking attempts to find a match for players in the queue
// Returns matched player entries if found, nil otherwise
func TryMatchmaking() (*QueueEntry, *QueueEntry, error) {
	if useMockRedis {
		matched, err := mocks.GetMockRedis().TryMatch()
		if err != nil || matched == nil || len(matched) < 2 {
			return nil, nil, err
		}
		p1 := QueueEntry{
			UserID:   matched[0].UserID,
			Name:     matched[0].Name,
			Picture:  matched[0].Picture,
			PodID:    matched[0].PodID,
			JoinedAt: matched[0].JoinedAt,
		}
		p2 := QueueEntry{
			UserID:   matched[1].UserID,
			Name:     matched[1].Name,
			Picture:  matched[1].Picture,
			PodID:    matched[1].PodID,
			JoinedAt: matched[1].JoinedAt,
		}
		return &p1, &p2, nil
	}

	if redisClient == nil {
		return nil, nil, fmt.Errorf("redis not initialized")
	}

	// Acquire distributed lock
	lockKey := matchmakingLockKey
	acquired, err := redisClient.SetNX(ctx, lockKey, podID, 2*time.Second).Result()
	if err != nil || !acquired {
		return nil, nil, nil // Another pod is handling matchmaking
	}
	defer redisClient.Del(ctx, lockKey)

	// Get first 2 players from queue
	entries, err := redisClient.ZRange(ctx, matchmakingQueueKey, 0, 1).Result()
	if err != nil {
		return nil, nil, err
	}

	if len(entries) < 2 {
		return nil, nil, nil // Not enough players
	}

	var player1, player2 QueueEntry
	if err := json.Unmarshal([]byte(entries[0]), &player1); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal([]byte(entries[1]), &player2); err != nil {
		return nil, nil, err
	}

	// Remove both players from queue
	redisClient.ZRem(ctx, matchmakingQueueKey, entries[0], entries[1])

	log.Printf("Matched players: %s vs %s", player1.UserID, player2.UserID)
	return &player1, &player2, nil
}

// PublishMatchNotification publishes a match notification to all pods
func PublishMatchNotification(match MatchNotification) error {
	if useMockRedis {
		return mocks.GetMockRedis().PublishMatch(mocks.MatchNotification{
			Player1ID: match.Player1ID,
			Player2ID: match.Player2ID,
			RoomID:    match.RoomID,
			HostPodID: match.HostPodID,
		})
	}

	if redisClient == nil {
		return fmt.Errorf("redis not initialized")
	}

	matchJSON, err := json.Marshal(match)
	if err != nil {
		return err
	}

	return redisClient.Publish(ctx, matchNotifyChannel, string(matchJSON)).Err()
}

// SubscribeToMatches subscribes to match notifications
func SubscribeToMatches(handler func(MatchNotification)) {
	if useMockRedis {
		ch := mocks.GetMockRedis().Subscribe()
		go func() {
			for msg := range ch {
				var match MatchNotification
				if err := json.Unmarshal([]byte(msg), &match); err != nil {
					continue
				}
				handler(match)
			}
		}()
		return
	}

	if redisClient == nil {
		log.Println("Redis not available, skipping match subscription")
		return
	}

	pubsub := redisClient.Subscribe(ctx, matchNotifyChannel)
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		var match MatchNotification
		if err := json.Unmarshal([]byte(msg.Payload), &match); err != nil {
			log.Printf("Failed to parse match notification: %v", err)
			continue
		}
		handler(match)
	}
}

// IsRedisAvailable returns true if Redis is connected and available (or mock mode is enabled)
func IsRedisAvailable() bool {
	// Mock mode is always "available"
	if useMockRedis {
		return true
	}

	if redisClient == nil {
		return false
	}
	_, err := redisClient.Ping(ctx).Result()
	return err == nil
}

// GetPodID returns the unique identifier for this pod
func GetPodID() string {
	return podID
}

// ==================== DISTRIBUTED GAME STATE ====================

// DistributedGameState stores the game state in Redis
type DistributedGameState struct {
	RoomID             string           `json:"roomId"`
	Player1ID          string           `json:"player1Id"`
	Player2ID          string           `json:"player2Id"`
	Player1Name        string           `json:"player1Name"`
	Player2Name        string           `json:"player2Name"`
	Player1Picture     string           `json:"player1Picture"`
	Player2Picture     string           `json:"player2Picture"`
	P1Score            int              `json:"p1Score"`
	P2Score            int              `json:"p2Score"`
	TimeRemaining      int              `json:"timeRemaining"`
	GoldenCookieActive bool             `json:"goldenCookieActive"`
	GoldenCookieX      float64          `json:"goldenCookieX"`
	GoldenCookieY      float64          `json:"goldenCookieY"`
	DoubleClickExpiry  map[string]int64 `json:"doubleClickExpiry"` // UserID -> Unix timestamp
	GameStarted        bool             `json:"gameStarted"`
	GameEnded          bool             `json:"gameEnded"`
	WinnerID           string           `json:"winnerId"`
	TimerPodID         string           `json:"timerPodId"` // Pod responsible for timer
}

// GameEvent represents an event that needs to be broadcast to all pods
type GameEvent struct {
	RoomID    string                 `json:"roomId"`
	EventType string                 `json:"eventType"`
	PlayerID  string                 `json:"playerId"`
	Data      map[string]interface{} `json:"data"`
}

const (
	gameStateKeyPrefix = "overcookied:game:"
	gameEventChannel   = "overcookied:game:events"
	gameStateTTL       = 10 * time.Minute
)

// Event types
const (
	EventGameStart   = "GAME_START"
	EventClick       = "CLICK"
	EventGoldenSpawn = "GOLDEN_SPAWN"
	EventGoldenClaim = "GOLDEN_CLAIM"
	EventStateUpdate = "STATE_UPDATE"
	EventGameEnd     = "GAME_END"
	EventPlayerQuit  = "PLAYER_QUIT"
)

// CreateDistributedGame creates a new game in Redis or mock store
func CreateDistributedGame(roomID string, p1, p2 *QueueEntry) error {
	state := DistributedGameState{
		RoomID:             roomID,
		Player1ID:          p1.UserID,
		Player2ID:          p2.UserID,
		Player1Name:        p1.Name,
		Player2Name:        p2.Name,
		Player1Picture:     p1.Picture,
		Player2Picture:     p2.Picture,
		P1Score:            0,
		P2Score:            0,
		TimeRemaining:      60,
		GoldenCookieActive: false,
		DoubleClickExpiry:  make(map[string]int64),
		GameStarted:        false,
		GameEnded:          false,
		TimerPodID:         podID,
	}

	return SaveGameState(&state)
}

// SaveGameState saves the game state to Redis or mock store
func SaveGameState(state *DistributedGameState) error {
	if useMockRedis {
		mockState := &mocks.GameState{
			RoomID:             state.RoomID,
			Player1ID:          state.Player1ID,
			Player2ID:          state.Player2ID,
			Player1Name:        state.Player1Name,
			Player2Name:        state.Player2Name,
			Player1Picture:     state.Player1Picture,
			Player2Picture:     state.Player2Picture,
			P1Score:            state.P1Score,
			P2Score:            state.P2Score,
			TimeRemaining:      state.TimeRemaining,
			GoldenCookieActive: state.GoldenCookieActive,
			GoldenCookieX:      state.GoldenCookieX,
			GoldenCookieY:      state.GoldenCookieY,
			DoubleClickExpiry:  state.DoubleClickExpiry,
			GameStarted:        state.GameStarted,
			GameEnded:          state.GameEnded,
			WinnerID:           state.WinnerID,
			TimerPodID:         state.TimerPodID,
		}
		return mocks.GetMockGameStore().SaveGameState(mockState)
	}

	if redisClient == nil {
		return fmt.Errorf("redis not initialized")
	}

	key := gameStateKeyPrefix + state.RoomID
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return redisClient.Set(ctx, key, string(stateJSON), gameStateTTL).Err()
}

// GetGameState retrieves the game state from Redis or mock store
func GetGameState(roomID string) (*DistributedGameState, error) {
	if useMockRedis {
		mockState, err := mocks.GetMockGameStore().GetGameState(roomID)
		if err != nil || mockState == nil {
			return nil, err
		}
		return &DistributedGameState{
			RoomID:             mockState.RoomID,
			Player1ID:          mockState.Player1ID,
			Player2ID:          mockState.Player2ID,
			Player1Name:        mockState.Player1Name,
			Player2Name:        mockState.Player2Name,
			Player1Picture:     mockState.Player1Picture,
			Player2Picture:     mockState.Player2Picture,
			P1Score:            mockState.P1Score,
			P2Score:            mockState.P2Score,
			TimeRemaining:      mockState.TimeRemaining,
			GoldenCookieActive: mockState.GoldenCookieActive,
			GoldenCookieX:      mockState.GoldenCookieX,
			GoldenCookieY:      mockState.GoldenCookieY,
			DoubleClickExpiry:  mockState.DoubleClickExpiry,
			GameStarted:        mockState.GameStarted,
			GameEnded:          mockState.GameEnded,
			WinnerID:           mockState.WinnerID,
			TimerPodID:         mockState.TimerPodID,
		}, nil
	}

	if redisClient == nil {
		return nil, fmt.Errorf("redis not initialized")
	}

	key := gameStateKeyPrefix + roomID
	stateJSON, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var state DistributedGameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// DeleteGameState removes the game state from Redis or mock store
func DeleteGameState(roomID string) error {
	if useMockRedis {
		return mocks.GetMockGameStore().DeleteGameState(roomID)
	}

	if redisClient == nil {
		return fmt.Errorf("redis not initialized")
	}

	key := gameStateKeyPrefix + roomID
	return redisClient.Del(ctx, key).Err()
}

// PublishGameEvent publishes a game event to all pods
func PublishGameEvent(event GameEvent) error {
	if useMockRedis {
		return mocks.GetMockGameStore().PublishGameEvent(mocks.GameEvent{
			RoomID:    event.RoomID,
			EventType: event.EventType,
			PlayerID:  event.PlayerID,
			Data:      event.Data,
		})
	}

	if redisClient == nil {
		return fmt.Errorf("redis not initialized")
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return redisClient.Publish(ctx, gameEventChannel, string(eventJSON)).Err()
}

// SubscribeToGameEvents subscribes to game events from all pods
func SubscribeToGameEvents(handler func(GameEvent)) {
	if useMockRedis {
		ch := mocks.GetMockGameStore().SubscribeToGameEvents()
		go func() {
			for mockEvent := range ch {
				event := GameEvent{
					RoomID:    mockEvent.RoomID,
					EventType: mockEvent.EventType,
					PlayerID:  mockEvent.PlayerID,
					Data:      mockEvent.Data,
				}
				handler(event)
			}
		}()
		return
	}

	if redisClient == nil {
		log.Println("Redis not available, skipping game event subscription")
		return
	}

	pubsub := redisClient.Subscribe(ctx, gameEventChannel)
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		var event GameEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			log.Printf("Failed to parse game event: %v", err)
			continue
		}
		handler(event)
	}
}

// AtomicScoreIncrement atomically increments a player's score
func AtomicScoreIncrement(roomID, playerID string, points int) (*DistributedGameState, error) {
	if useMockRedis {
		mockState, err := mocks.GetMockGameStore().GetGameState(roomID)
		if err != nil || mockState == nil {
			return nil, fmt.Errorf("game not found: %s", roomID)
		}

		if playerID == mockState.Player1ID {
			mockState.P1Score += points
		} else if playerID == mockState.Player2ID {
			mockState.P2Score += points
		}

		mocks.GetMockGameStore().SaveGameState(mockState)

		return &DistributedGameState{
			RoomID:             mockState.RoomID,
			Player1ID:          mockState.Player1ID,
			Player2ID:          mockState.Player2ID,
			Player1Name:        mockState.Player1Name,
			Player2Name:        mockState.Player2Name,
			Player1Picture:     mockState.Player1Picture,
			Player2Picture:     mockState.Player2Picture,
			P1Score:            mockState.P1Score,
			P2Score:            mockState.P2Score,
			TimeRemaining:      mockState.TimeRemaining,
			GoldenCookieActive: mockState.GoldenCookieActive,
			GoldenCookieX:      mockState.GoldenCookieX,
			GoldenCookieY:      mockState.GoldenCookieY,
			DoubleClickExpiry:  mockState.DoubleClickExpiry,
			GameStarted:        mockState.GameStarted,
			GameEnded:          mockState.GameEnded,
			WinnerID:           mockState.WinnerID,
			TimerPodID:         mockState.TimerPodID,
		}, nil
	}

	if redisClient == nil {
		return nil, fmt.Errorf("redis not initialized")
	}

	// Use Redis transaction for atomic update
	key := gameStateKeyPrefix + roomID

	var updatedState *DistributedGameState

	err := redisClient.Watch(ctx, func(tx *redis.Tx) error {
		stateJSON, err := tx.Get(ctx, key).Result()
		if err != nil {
			return err
		}

		var state DistributedGameState
		if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
			return err
		}

		// Update score
		if playerID == state.Player1ID {
			state.P1Score += points
		} else if playerID == state.Player2ID {
			state.P2Score += points
		}

		// Save back
		newStateJSON, err := json.Marshal(state)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, string(newStateJSON), gameStateTTL)
			return nil
		})

		updatedState = &state
		return err
	}, key)

	return updatedState, err
}

// AtomicClaimGoldenCookie atomically claims the golden cookie
func AtomicClaimGoldenCookie(roomID, playerID string) (bool, error) {
	if useMockRedis {
		mockState, err := mocks.GetMockGameStore().GetGameState(roomID)
		if err != nil || mockState == nil {
			return false, fmt.Errorf("game not found: %s", roomID)
		}

		if !mockState.GoldenCookieActive {
			return false, nil
		}

		mockState.GoldenCookieActive = false
		if mockState.DoubleClickExpiry == nil {
			mockState.DoubleClickExpiry = make(map[string]int64)
		}
		mockState.DoubleClickExpiry[playerID] = time.Now().Add(3 * time.Second).Unix()
		mocks.GetMockGameStore().SaveGameState(mockState)
		return true, nil
	}

	if redisClient == nil {
		return false, fmt.Errorf("redis not initialized")
	}

	key := gameStateKeyPrefix + roomID
	claimed := false

	err := redisClient.Watch(ctx, func(tx *redis.Tx) error {
		stateJSON, err := tx.Get(ctx, key).Result()
		if err != nil {
			return err
		}

		var state DistributedGameState
		if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
			return err
		}

		// Check if golden cookie is still active
		if !state.GoldenCookieActive {
			claimed = false
			return nil
		}

		// Claim it!
		state.GoldenCookieActive = false
		state.DoubleClickExpiry[playerID] = time.Now().Add(3 * time.Second).Unix()
		claimed = true

		// Save back
		newStateJSON, err := json.Marshal(state)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, string(newStateJSON), gameStateTTL)
			return nil
		})

		return err
	}, key)

	return claimed, err
}
