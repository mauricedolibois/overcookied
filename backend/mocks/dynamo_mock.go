package mocks

import (
	"log"
	"sort"
	"sync"
	"time"
)

// MockDynamoDB provides an in-memory mock for DynamoDB operations
type MockDynamoDB struct {
	mu    sync.RWMutex
	users map[string]CookieUser
	games []CookieGame
}

// CookieUser represents a user in the mock database
type CookieUser struct {
	UserID  string `json:"userId"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Score   int    `json:"score"`
}

// CookieGame represents a game record in the mock database
type CookieGame struct {
	GameID          string `json:"gameId"`
	PlayerID        string `json:"playerId"`
	Timestamp       int64  `json:"timestamp"`
	Score           int    `json:"score"`
	OpponentScore   int    `json:"opponentScore"`
	Reason          string `json:"reason"`
	Won             bool   `json:"won"`
	WinnerID        string `json:"winnerId"`
	Opponent        string `json:"opponent"`
	PlayerName      string `json:"playerName"`
	PlayerPicture   string `json:"playerPicture"`
	OpponentName    string `json:"opponentName"`
	OpponentPicture string `json:"opponentPicture"`
}

var mockDynamoInstance *MockDynamoDB
var mockDynamoOnce sync.Once

// GetMockDynamoDB returns the singleton mock DynamoDB instance
func GetMockDynamoDB() *MockDynamoDB {
	mockDynamoOnce.Do(func() {
		mockDynamoInstance = &MockDynamoDB{
			users: make(map[string]CookieUser),
			games: make([]CookieGame, 0),
		}
		// Add some sample data for local development
		mockDynamoInstance.seedData()
		log.Println("[MOCK] In-memory DynamoDB initialized for local development")
	})
	return mockDynamoInstance
}

// seedData adds sample data for local testing
func (m *MockDynamoDB) seedData() {
	// Sample users
	sampleUsers := []CookieUser{
		{UserID: "mock-user-1", Email: "alice@example.com", Name: "Alice Baker", Picture: "", Score: 1500},
		{UserID: "mock-user-2", Email: "bob@example.com", Name: "Bob Chef", Picture: "", Score: 1200},
		{UserID: "mock-user-3", Email: "charlie@example.com", Name: "Charlie Cook", Picture: "", Score: 900},
	}
	for _, u := range sampleUsers {
		m.users[u.UserID] = u
	}

	// Sample games
	now := time.Now().Unix()
	sampleGames := []CookieGame{
		{
			GameID: "mock-game-1", PlayerID: "mock-user-1", Timestamp: now - 3600,
			Score: 150, OpponentScore: 120, Won: true, WinnerID: "mock-user-1",
			Opponent: "mock-user-2", PlayerName: "Alice Baker", OpponentName: "Bob Chef",
			Reason: "time_up",
		},
		{
			GameID: "mock-game-2", PlayerID: "mock-user-2", Timestamp: now - 3600,
			Score: 120, OpponentScore: 150, Won: false, WinnerID: "mock-user-1",
			Opponent: "mock-user-1", PlayerName: "Bob Chef", OpponentName: "Alice Baker",
			Reason: "time_up",
		},
		{
			GameID: "mock-game-3", PlayerID: "mock-user-1", Timestamp: now - 7200,
			Score: 200, OpponentScore: 180, Won: true, WinnerID: "mock-user-1",
			Opponent: "mock-user-3", PlayerName: "Alice Baker", OpponentName: "Charlie Cook",
			Reason: "time_up",
		},
	}
	m.games = append(m.games, sampleGames...)

	log.Printf("[MOCK] Seeded %d users and %d games for local development", len(sampleUsers), len(sampleGames))
}

// --- User Operations ---

// SaveUser saves or updates a user
func (m *MockDynamoDB) SaveUser(user CookieUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.users[user.UserID]
	if exists {
		// Preserve score when updating
		user.Score = existing.Score
	}
	m.users[user.UserID] = user
	log.Printf("[MOCK] User saved: %s (%s)", user.Name, user.UserID)
	return nil
}

// GetUser retrieves a user by ID
func (m *MockDynamoDB) GetUser(userID string) (*CookieUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil, nil
	}
	return &user, nil
}

// GetTopUsers returns the top users by score
func (m *MockDynamoDB) GetTopUsers(limit int) ([]CookieUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]CookieUser, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, u)
	}

	// Sort by score descending
	sort.Slice(users, func(i, j int) bool {
		return users[i].Score > users[j].Score
	})

	if limit > len(users) {
		limit = len(users)
	}
	return users[:limit], nil
}

// IncrementUserScore increments a user's score
func (m *MockDynamoDB) IncrementUserScore(userID string, delta int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return nil
	}
	user.Score += delta
	m.users[userID] = user
	log.Printf("[MOCK] User %s score updated: +%d (total: %d)", userID, delta, user.Score)
	return nil
}

// --- Game Operations ---

// SaveGame saves a game record
func (m *MockDynamoDB) SaveGame(game CookieGame) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.games = append(m.games, game)
	log.Printf("[MOCK] Game saved: %s (Player: %s, Score: %d)", game.GameID, game.PlayerID, game.Score)
	return nil
}

// GetGamesByPlayer returns games for a specific player, sorted by timestamp descending
func (m *MockDynamoDB) GetGamesByPlayer(playerID string, limit int) ([]CookieGame, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	playerGames := make([]CookieGame, 0)
	for _, g := range m.games {
		if g.PlayerID == playerID {
			playerGames = append(playerGames, g)
		}
	}

	// Sort by timestamp descending
	sort.Slice(playerGames, func(i, j int) bool {
		return playerGames[i].Timestamp > playerGames[j].Timestamp
	})

	if limit > len(playerGames) {
		limit = len(playerGames)
	}
	return playerGames[:limit], nil
}

// GetRecentGames returns the most recent games across all players
func (m *MockDynamoDB) GetRecentGames(limit int) ([]CookieGame, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	games := make([]CookieGame, len(m.games))
	copy(games, m.games)

	// Sort by timestamp descending
	sort.Slice(games, func(i, j int) bool {
		return games[i].Timestamp > games[j].Timestamp
	})

	if limit > len(games) {
		limit = len(games)
	}
	return games[:limit], nil
}

// GetUserStats returns basic stats for a user
func (m *MockDynamoDB) GetUserStats(userID string) (wins int, losses int, totalGames int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, g := range m.games {
		if g.PlayerID == userID {
			totalGames++
			if g.Won {
				wins++
			} else {
				losses++
			}
		}
	}
	return
}

// CountGamesByPlayer returns the total number of games for a player
func (m *MockDynamoDB) CountGamesByPlayer(playerID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, g := range m.games {
		if g.PlayerID == playerID {
			count++
		}
	}
	return count
}
