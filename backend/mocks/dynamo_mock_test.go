package mocks

import (
	"sync"
	"testing"
)

// resetMockDynamoDB creates a fresh MockDynamoDB instance for testing
func newTestMockDynamoDB() *MockDynamoDB {
	return &MockDynamoDB{
		users: make(map[string]CookieUser),
		games: make([]CookieGame, 0),
	}
}

func TestSaveAndGetUser(t *testing.T) {
	db := newTestMockDynamoDB()

	user := CookieUser{
		UserID:  "test-user-1",
		Email:   "test@example.com",
		Name:    "Test User",
		Picture: "https://example.com/pic.jpg",
		Score:   100,
	}

	// Save user
	err := db.SaveUser(user)
	if err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	// Retrieve user
	retrieved, err := db.GetUser("test-user-1")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetUser returned nil for existing user")
	}

	if retrieved.UserID != user.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", retrieved.UserID, user.UserID)
	}
	if retrieved.Email != user.Email {
		t.Errorf("Email mismatch: got %s, want %s", retrieved.Email, user.Email)
	}
	if retrieved.Name != user.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, user.Name)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	db := newTestMockDynamoDB()

	retrieved, err := db.GetUser("non-existent-user")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil for non-existent user, got %+v", retrieved)
	}
}

func TestSaveUser_PreservesScoreOnUpdate(t *testing.T) {
	db := newTestMockDynamoDB()

	// Create initial user with score
	user := CookieUser{
		UserID:  "test-user-1",
		Email:   "test@example.com",
		Name:    "Test User",
		Picture: "https://example.com/pic.jpg",
		Score:   500,
	}
	db.SaveUser(user)

	// Update user profile (simulating login update)
	updatedUser := CookieUser{
		UserID:  "test-user-1",
		Email:   "test@example.com",
		Name:    "New Name",
		Picture: "https://example.com/new-pic.jpg",
		Score:   0, // Score would be 0 from login data
	}
	db.SaveUser(updatedUser)

	// Verify score is preserved
	retrieved, _ := db.GetUser("test-user-1")
	if retrieved.Score != 500 {
		t.Errorf("Score was not preserved: got %d, want 500", retrieved.Score)
	}
	if retrieved.Name != "New Name" {
		t.Errorf("Name was not updated: got %s, want 'New Name'", retrieved.Name)
	}
}

func TestGetTopUsers(t *testing.T) {
	db := newTestMockDynamoDB()

	// Add users with different scores
	users := []CookieUser{
		{UserID: "user1", Name: "User One", Score: 100},
		{UserID: "user2", Name: "User Two", Score: 500},
		{UserID: "user3", Name: "User Three", Score: 250},
		{UserID: "user4", Name: "User Four", Score: 750},
		{UserID: "user5", Name: "User Five", Score: 50},
	}
	for _, u := range users {
		db.users[u.UserID] = u
	}

	// Get top 3
	topUsers, err := db.GetTopUsers(3)
	if err != nil {
		t.Fatalf("GetTopUsers failed: %v", err)
	}

	if len(topUsers) != 3 {
		t.Fatalf("Expected 3 users, got %d", len(topUsers))
	}

	// Verify sorted by score descending
	expectedOrder := []string{"user4", "user2", "user3"} // 750, 500, 250
	for i, expected := range expectedOrder {
		if topUsers[i].UserID != expected {
			t.Errorf("Position %d: expected %s, got %s", i, expected, topUsers[i].UserID)
		}
	}
}

func TestGetTopUsers_LimitExceedsTotal(t *testing.T) {
	db := newTestMockDynamoDB()

	db.users["user1"] = CookieUser{UserID: "user1", Score: 100}
	db.users["user2"] = CookieUser{UserID: "user2", Score: 200}

	topUsers, err := db.GetTopUsers(10)
	if err != nil {
		t.Fatalf("GetTopUsers failed: %v", err)
	}

	if len(topUsers) != 2 {
		t.Errorf("Expected 2 users (all available), got %d", len(topUsers))
	}
}

func TestIncrementUserScore(t *testing.T) {
	db := newTestMockDynamoDB()

	db.users["user1"] = CookieUser{UserID: "user1", Name: "Test", Score: 100}

	// Increment score
	err := db.IncrementUserScore("user1", 50)
	if err != nil {
		t.Fatalf("IncrementUserScore failed: %v", err)
	}

	user, _ := db.GetUser("user1")
	if user.Score != 150 {
		t.Errorf("Score not incremented correctly: got %d, want 150", user.Score)
	}
}

func TestIncrementUserScore_NegativeDelta(t *testing.T) {
	db := newTestMockDynamoDB()

	db.users["user1"] = CookieUser{UserID: "user1", Score: 100}

	// Decrement score (negative delta)
	db.IncrementUserScore("user1", -30)

	user, _ := db.GetUser("user1")
	if user.Score != 70 {
		t.Errorf("Score not decremented correctly: got %d, want 70", user.Score)
	}
}

func TestSaveAndGetGame(t *testing.T) {
	db := newTestMockDynamoDB()

	game := CookieGame{
		GameID:        "game-123",
		PlayerID:      "player-1",
		Timestamp:     1706500000,
		Score:         150,
		OpponentScore: 120,
		Won:           true,
		WinnerID:      "player-1",
		Opponent:      "player-2",
		Reason:        "time_up",
	}

	err := db.SaveGame(game)
	if err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	games, err := db.GetGamesByPlayer("player-1", 10)
	if err != nil {
		t.Fatalf("GetGamesByPlayer failed: %v", err)
	}

	if len(games) != 1 {
		t.Fatalf("Expected 1 game, got %d", len(games))
	}

	if games[0].GameID != "game-123" {
		t.Errorf("GameID mismatch: got %s, want game-123", games[0].GameID)
	}
	if games[0].Score != 150 {
		t.Errorf("Score mismatch: got %d, want 150", games[0].Score)
	}
}

func TestGetGamesByPlayer_SortedByTimestamp(t *testing.T) {
	db := newTestMockDynamoDB()

	// Add games in random order
	games := []CookieGame{
		{GameID: "game-1", PlayerID: "player-1", Timestamp: 1000},
		{GameID: "game-3", PlayerID: "player-1", Timestamp: 3000},
		{GameID: "game-2", PlayerID: "player-1", Timestamp: 2000},
	}
	for _, g := range games {
		db.SaveGame(g)
	}

	retrieved, _ := db.GetGamesByPlayer("player-1", 10)

	// Should be sorted descending (newest first)
	if retrieved[0].GameID != "game-3" {
		t.Errorf("Expected newest game first, got %s", retrieved[0].GameID)
	}
	if retrieved[2].GameID != "game-1" {
		t.Errorf("Expected oldest game last, got %s", retrieved[2].GameID)
	}
}

func TestCountGamesByPlayer(t *testing.T) {
	db := newTestMockDynamoDB()

	// Add games for multiple players
	db.games = []CookieGame{
		{GameID: "game-1", PlayerID: "player-1"},
		{GameID: "game-2", PlayerID: "player-1"},
		{GameID: "game-3", PlayerID: "player-2"},
		{GameID: "game-4", PlayerID: "player-1"},
	}

	count := db.CountGamesByPlayer("player-1")
	if count != 3 {
		t.Errorf("Expected 3 games for player-1, got %d", count)
	}

	count2 := db.CountGamesByPlayer("player-2")
	if count2 != 1 {
		t.Errorf("Expected 1 game for player-2, got %d", count2)
	}
}

func TestGetUserStats(t *testing.T) {
	db := newTestMockDynamoDB()

	db.games = []CookieGame{
		{GameID: "game-1", PlayerID: "player-1", Won: true},
		{GameID: "game-2", PlayerID: "player-1", Won: false},
		{GameID: "game-3", PlayerID: "player-1", Won: true},
		{GameID: "game-4", PlayerID: "player-1", Won: true},
		{GameID: "game-5", PlayerID: "player-2", Won: true}, // Different player
	}

	wins, losses, total := db.GetUserStats("player-1")

	if wins != 3 {
		t.Errorf("Expected 3 wins, got %d", wins)
	}
	if losses != 1 {
		t.Errorf("Expected 1 loss, got %d", losses)
	}
	if total != 4 {
		t.Errorf("Expected 4 total games, got %d", total)
	}
}

func TestConcurrentUserOperations(t *testing.T) {
	db := newTestMockDynamoDB()

	// Seed initial user
	db.users["concurrent-user"] = CookieUser{UserID: "concurrent-user", Score: 0}

	var wg sync.WaitGroup
	iterations := 100

	// Concurrently increment score
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db.IncrementUserScore("concurrent-user", 1)
		}()
	}

	wg.Wait()

	user, _ := db.GetUser("concurrent-user")
	if user.Score != iterations {
		t.Errorf("Concurrent score updates failed: got %d, want %d", user.Score, iterations)
	}
}
