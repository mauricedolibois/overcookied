package mocks

import (
	"sync"
	"testing"
	"time"
)

// newTestMockRedis creates a fresh MockRedis instance for testing
func newTestMockRedis() *MockRedis {
	return &MockRedis{
		queue:       make([]QueueEntry, 0),
		pubsubChan:  make(chan string, 100),
		subscribers: make([]chan string, 0),
		podID:       "test-pod",
	}
}

func TestAddToQueue(t *testing.T) {
	redis := newTestMockRedis()

	err := redis.AddToQueue("user-1", "Player One", "https://pic.url")
	if err != nil {
		t.Fatalf("AddToQueue failed: %v", err)
	}

	length := redis.GetQueueLength()
	if length != 1 {
		t.Errorf("Expected queue length 1, got %d", length)
	}

	entries := redis.GetQueueEntries()
	if entries[0].UserID != "user-1" {
		t.Errorf("Expected user-1, got %s", entries[0].UserID)
	}
	if entries[0].Name != "Player One" {
		t.Errorf("Expected Player One, got %s", entries[0].Name)
	}
}

func TestAddToQueue_ReplacesExistingUser(t *testing.T) {
	redis := newTestMockRedis()

	// Add user first time
	redis.AddToQueue("user-1", "Old Name", "old-pic")

	// Wait a bit so timestamp differs
	time.Sleep(10 * time.Millisecond)

	// Add same user again with different data
	redis.AddToQueue("user-1", "New Name", "new-pic")

	// Should still be 1 entry, not 2
	length := redis.GetQueueLength()
	if length != 1 {
		t.Errorf("Expected queue length 1 (replaced), got %d", length)
	}

	entries := redis.GetQueueEntries()
	if entries[0].Name != "New Name" {
		t.Errorf("Expected updated name 'New Name', got %s", entries[0].Name)
	}
}

func TestRemoveFromQueue(t *testing.T) {
	redis := newTestMockRedis()

	redis.AddToQueue("user-1", "Player One", "")
	redis.AddToQueue("user-2", "Player Two", "")

	err := redis.RemoveFromQueue("user-1")
	if err != nil {
		t.Fatalf("RemoveFromQueue failed: %v", err)
	}

	length := redis.GetQueueLength()
	if length != 1 {
		t.Errorf("Expected queue length 1 after removal, got %d", length)
	}

	entries := redis.GetQueueEntries()
	if entries[0].UserID != "user-2" {
		t.Errorf("Expected user-2 to remain, got %s", entries[0].UserID)
	}
}

func TestRemoveFromQueue_NonExistent(t *testing.T) {
	redis := newTestMockRedis()

	redis.AddToQueue("user-1", "Player One", "")

	// Should not error when removing non-existent user
	err := redis.RemoveFromQueue("non-existent")
	if err != nil {
		t.Fatalf("RemoveFromQueue should not error for non-existent user: %v", err)
	}

	// Queue should be unchanged
	length := redis.GetQueueLength()
	if length != 1 {
		t.Errorf("Queue should remain unchanged, got length %d", length)
	}
}

func TestGetQueueLength(t *testing.T) {
	redis := newTestMockRedis()

	if redis.GetQueueLength() != 0 {
		t.Errorf("Empty queue should have length 0")
	}

	redis.AddToQueue("user-1", "One", "")
	redis.AddToQueue("user-2", "Two", "")
	redis.AddToQueue("user-3", "Three", "")

	if redis.GetQueueLength() != 3 {
		t.Errorf("Expected queue length 3, got %d", redis.GetQueueLength())
	}
}

func TestTryMatch_SuccessfulMatch(t *testing.T) {
	redis := newTestMockRedis()

	redis.AddToQueue("user-1", "Player One", "pic1")
	redis.AddToQueue("user-2", "Player Two", "pic2")

	matched, err := redis.TryMatch()
	if err != nil {
		t.Fatalf("TryMatch failed: %v", err)
	}

	if matched == nil {
		t.Fatal("Expected matched players, got nil")
	}

	if len(matched) != 2 {
		t.Fatalf("Expected 2 matched players, got %d", len(matched))
	}

	// Verify FIFO order (first in queue matched first)
	if matched[0].UserID != "user-1" {
		t.Errorf("Expected first player to be user-1, got %s", matched[0].UserID)
	}
	if matched[1].UserID != "user-2" {
		t.Errorf("Expected second player to be user-2, got %s", matched[1].UserID)
	}

	// Queue should be empty after match
	if redis.GetQueueLength() != 0 {
		t.Errorf("Queue should be empty after match, got length %d", redis.GetQueueLength())
	}
}

func TestTryMatch_NotEnoughPlayers(t *testing.T) {
	redis := newTestMockRedis()

	redis.AddToQueue("user-1", "Player One", "")

	matched, err := redis.TryMatch()
	if err != nil {
		t.Fatalf("TryMatch failed: %v", err)
	}

	if matched != nil {
		t.Errorf("Expected nil when not enough players, got %+v", matched)
	}

	// Player should still be in queue
	if redis.GetQueueLength() != 1 {
		t.Errorf("Player should remain in queue when no match, got length %d", redis.GetQueueLength())
	}
}

func TestTryMatch_QueueOrderMaintained(t *testing.T) {
	redis := newTestMockRedis()

	// Add 4 players
	redis.AddToQueue("user-1", "One", "")
	redis.AddToQueue("user-2", "Two", "")
	redis.AddToQueue("user-3", "Three", "")
	redis.AddToQueue("user-4", "Four", "")

	// First match should get first two players
	matched1, _ := redis.TryMatch()
	if matched1[0].UserID != "user-1" || matched1[1].UserID != "user-2" {
		t.Errorf("First match should be user-1 and user-2")
	}

	// Second match should get remaining players
	matched2, _ := redis.TryMatch()
	if matched2[0].UserID != "user-3" || matched2[1].UserID != "user-4" {
		t.Errorf("Second match should be user-3 and user-4")
	}
}

func TestPublishAndSubscribe(t *testing.T) {
	redis := newTestMockRedis()

	// Subscribe to matches
	subscriber := redis.Subscribe()

	// Publish a match notification
	notification := MatchNotification{
		Player1ID: "player-1",
		Player2ID: "player-2",
		RoomID:    "room-123",
		HostPodID: "pod-1",
	}

	err := redis.PublishMatch(notification)
	if err != nil {
		t.Fatalf("PublishMatch failed: %v", err)
	}

	// Receive the notification
	select {
	case msg := <-subscriber:
		if msg == "" {
			t.Error("Received empty message")
		}
		// Message should contain the room ID
		if len(msg) < 10 {
			t.Errorf("Message too short: %s", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for match notification")
	}
}

func TestConcurrentQueueOperations(t *testing.T) {
	redis := newTestMockRedis()

	var wg sync.WaitGroup
	numUsers := 50

	// Concurrently add users
	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			redis.AddToQueue(
				string(rune('a'+id%26))+string(rune('0'+id/26)),
				"Player",
				"",
			)
		}(i)
	}

	wg.Wait()

	// All users should be in queue (might be less due to duplicate handling)
	length := redis.GetQueueLength()
	if length == 0 {
		t.Error("Queue should not be empty after concurrent adds")
	}
}

func TestGetQueueEntries_ReturnsCopy(t *testing.T) {
	redis := newTestMockRedis()

	redis.AddToQueue("user-1", "One", "")

	entries := redis.GetQueueEntries()

	// Modifying returned slice should not affect internal state
	entries[0].Name = "Modified"

	internalEntries := redis.GetQueueEntries()
	if internalEntries[0].Name == "Modified" {
		t.Error("GetQueueEntries should return a copy, not the internal slice")
	}
}

// Test GameStore functionality

func newTestMockGameStore() *MockGameStore {
	return &MockGameStore{
		games:            make(map[string]*GameState),
		eventSubscribers: make([]chan GameEvent, 0),
	}
}

func TestSaveAndGetGameState(t *testing.T) {
	store := newTestMockGameStore()

	state := &GameState{
		RoomID:        "room-123",
		Player1ID:     "player-1",
		Player2ID:     "player-2",
		P1Score:       50,
		P2Score:       45,
		TimeRemaining: 30,
	}

	err := store.SaveGameState(state)
	if err != nil {
		t.Fatalf("SaveGameState failed: %v", err)
	}

	retrieved, err := store.GetGameState("room-123")
	if err != nil {
		t.Fatalf("GetGameState failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected game state, got nil")
	}

	if retrieved.P1Score != 50 {
		t.Errorf("P1Score mismatch: got %d, want 50", retrieved.P1Score)
	}
	if retrieved.Player1ID != "player-1" {
		t.Errorf("Player1ID mismatch: got %s, want player-1", retrieved.Player1ID)
	}
}

func TestDeleteGameState(t *testing.T) {
	store := newTestMockGameStore()

	state := &GameState{RoomID: "room-to-delete"}
	store.SaveGameState(state)

	err := store.DeleteGameState("room-to-delete")
	if err != nil {
		t.Fatalf("DeleteGameState failed: %v", err)
	}

	retrieved, _ := store.GetGameState("room-to-delete")
	if retrieved != nil {
		t.Error("Game state should be deleted")
	}
}

func TestGameEventPubSub(t *testing.T) {
	store := newTestMockGameStore()

	subscriber := store.SubscribeToGameEvents()

	event := GameEvent{
		RoomID:    "room-123",
		EventType: "CLICK",
		PlayerID:  "player-1",
		Data:      map[string]interface{}{"points": 1},
	}

	store.PublishGameEvent(event)

	select {
	case received := <-subscriber:
		if received.RoomID != "room-123" {
			t.Errorf("RoomID mismatch: got %s, want room-123", received.RoomID)
		}
		if received.EventType != "CLICK" {
			t.Errorf("EventType mismatch: got %s, want CLICK", received.EventType)
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for game event")
	}
}
