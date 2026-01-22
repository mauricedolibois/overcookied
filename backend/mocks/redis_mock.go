package mocks

import (
	"encoding/json"
	"log"
	"sort"
	"sync"
	"time"
)

// QueueTTL is the time after which a player is automatically removed from the queue
const QueueTTL = 30 * time.Second

// MockRedis provides an in-memory mock for Redis/Valkey operations
type MockRedis struct {
	mu          sync.RWMutex
	queue       []QueueEntry
	pubsubChan  chan string
	subscribers []chan string
	podID       string
}

// QueueEntry represents a player in the matchmaking queue
type QueueEntry struct {
	UserID   string `json:"userId"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	PodID    string `json:"podId"`
	JoinedAt int64  `json:"joinedAt"`
}

// MatchNotification is sent when a match is found
type MatchNotification struct {
	Player1ID string `json:"player1Id"`
	Player2ID string `json:"player2Id"`
	RoomID    string `json:"roomId"`
	HostPodID string `json:"hostPodId"`
}

var mockRedisInstance *MockRedis
var mockRedisOnce sync.Once

// GetMockRedis returns the singleton mock redis instance
func GetMockRedis() *MockRedis {
	mockRedisOnce.Do(func() {
		mockRedisInstance = &MockRedis{
			queue:       make([]QueueEntry, 0),
			pubsubChan:  make(chan string, 100),
			subscribers: make([]chan string, 0),
			podID:       "mock-pod-local",
		}
		// Start background cleanup goroutine to remove stale entries
		go mockRedisInstance.cleanupStaleEntries()
		log.Println("[MOCK] In-memory Redis/Valkey initialized for local development")
	})
	return mockRedisInstance
}

// cleanupStaleEntries periodically removes entries older than QueueTTL
func (m *MockRedis) cleanupStaleEntries() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now().Unix()
		ttlSeconds := int64(QueueTTL.Seconds())

		// Filter out stale entries
		newQueue := make([]QueueEntry, 0, len(m.queue))
		for _, entry := range m.queue {
			if now-entry.JoinedAt < ttlSeconds {
				newQueue = append(newQueue, entry)
			} else {
				log.Printf("[MOCK] Removed stale player from queue: %s (joined %ds ago)",
					entry.UserID, now-entry.JoinedAt)
			}
		}
		m.queue = newQueue
		m.mu.Unlock()
	}
}

// GetPodID returns the mock pod ID
func (m *MockRedis) GetPodID() string {
	return m.podID
}

// AddToQueue adds a player to the mock matchmaking queue
func (m *MockRedis) AddToQueue(userID, name, picture string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove existing entry for this user (if rejoining)
	for i, entry := range m.queue {
		if entry.UserID == userID {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			break
		}
	}

	entry := QueueEntry{
		UserID:   userID,
		Name:     name,
		Picture:  picture,
		PodID:    m.podID,
		JoinedAt: time.Now().Unix(),
	}
	m.queue = append(m.queue, entry)

	// Sort by JoinedAt (oldest first) - mimics Redis Sorted Set behavior
	sort.Slice(m.queue, func(i, j int) bool {
		return m.queue[i].JoinedAt < m.queue[j].JoinedAt
	})

	log.Printf("[MOCK] Player added to queue: %s (%s) - Queue size: %d", name, userID, len(m.queue))
	return nil
}

// RemoveFromQueue removes a player from the mock queue
func (m *MockRedis) RemoveFromQueue(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, entry := range m.queue {
		if entry.UserID == userID {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			log.Printf("[MOCK] Player removed from queue: %s - Queue size: %d", userID, len(m.queue))
			break
		}
	}
	return nil
}

// GetQueueLength returns the number of players in the mock queue
func (m *MockRedis) GetQueueLength() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.queue))
}

// TryMatch attempts to match two players from the queue
// Returns the matched players or nil if not enough players
// Mimics the distributed lock behavior - but since we're single-pod, no lock needed
func (m *MockRedis) TryMatch() ([]QueueEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove stale entries first (entries older than TTL)
	now := time.Now().Unix()
	ttlSeconds := int64(QueueTTL.Seconds())
	newQueue := make([]QueueEntry, 0, len(m.queue))
	for _, entry := range m.queue {
		if now-entry.JoinedAt < ttlSeconds {
			newQueue = append(newQueue, entry)
		}
	}
	m.queue = newQueue

	if len(m.queue) < 2 {
		return nil, nil
	}

	// Get first two players (oldest in queue - FIFO)
	matched := []QueueEntry{m.queue[0], m.queue[1]}
	m.queue = m.queue[2:]

	log.Printf("[MOCK] Match found: %s vs %s - Queue size: %d", matched[0].Name, matched[1].Name, len(m.queue))
	return matched, nil
}

// PublishMatch sends a match notification
func (m *MockRedis) PublishMatch(notification MatchNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// Send to all subscribers
	for _, sub := range m.subscribers {
		select {
		case sub <- string(data):
		default:
			// Channel full, skip
		}
	}

	log.Printf("[MOCK] Match published: %s vs %s in room %s",
		notification.Player1ID, notification.Player2ID, notification.RoomID)
	return nil
}

// Subscribe returns a channel for match notifications
func (m *MockRedis) Subscribe() chan string {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan string, 10)
	m.subscribers = append(m.subscribers, ch)
	return ch
}

// GetQueueEntries returns all current queue entries (for debugging)
func (m *MockRedis) GetQueueEntries() []QueueEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := make([]QueueEntry, len(m.queue))
	copy(entries, m.queue)
	return entries
}

// ==================== GAME STATE MOCK ====================

// GameState stores the game state in memory
type GameState struct {
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
	DoubleClickExpiry  map[string]int64 `json:"doubleClickExpiry"`
	GameStarted        bool             `json:"gameStarted"`
	GameEnded          bool             `json:"gameEnded"`
	WinnerID           string           `json:"winnerId"`
	TimerPodID         string           `json:"timerPodId"`
}

// GameEvent represents a game event
type GameEvent struct {
	RoomID    string                 `json:"roomId"`
	EventType string                 `json:"eventType"`
	PlayerID  string                 `json:"playerId"`
	Data      map[string]interface{} `json:"data"`
}

// MockGameStore stores game states in memory
type MockGameStore struct {
	mu               sync.RWMutex
	games            map[string]*GameState
	eventSubscribers []chan GameEvent
}

var mockGameStoreInstance *MockGameStore
var mockGameStoreOnce sync.Once

// GetMockGameStore returns the singleton mock game store
func GetMockGameStore() *MockGameStore {
	mockGameStoreOnce.Do(func() {
		mockGameStoreInstance = &MockGameStore{
			games:            make(map[string]*GameState),
			eventSubscribers: make([]chan GameEvent, 0),
		}
		log.Println("[MOCK] In-memory game store initialized")
	})
	return mockGameStoreInstance
}

// SaveGameState saves a game state
func (s *MockGameStore) SaveGameState(state *GameState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.games[state.RoomID] = state
	log.Printf("[MOCK] Game state saved: %s", state.RoomID)
	return nil
}

// GetGameState retrieves a game state
func (s *MockGameStore) GetGameState(roomID string) (*GameState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, exists := s.games[roomID]
	if !exists {
		return nil, nil
	}
	return state, nil
}

// DeleteGameState deletes a game state
func (s *MockGameStore) DeleteGameState(roomID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.games, roomID)
	log.Printf("[MOCK] Game state deleted: %s", roomID)
	return nil
}

// PublishGameEvent publishes a game event to all subscribers
func (s *MockGameStore) PublishGameEvent(event GameEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sub := range s.eventSubscribers {
		select {
		case sub <- event:
		default:
			// Channel full, skip
		}
	}
	return nil
}

// SubscribeToGameEvents returns a channel for game events
func (s *MockGameStore) SubscribeToGameEvents() chan GameEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan GameEvent, 100)
	s.eventSubscribers = append(s.eventSubscribers, ch)
	return ch
}
