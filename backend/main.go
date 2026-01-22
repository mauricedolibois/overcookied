package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mauricedolibois/overcookied/backend/db"
)

// Response structure for API endpoints
type Response struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Message: "Backend is running",
		Status:  "healthy",
	}
	json.NewEncoder(w).Encode(response)
}

// API info endpoint
func apiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Message: "Cookie Clicker API",
		Status:  "ready",
	}
	json.NewEncoder(w).Encode(response)
}

func enableCors(w *http.ResponseWriter) {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	// Remove trailing slash if present
	frontendURL = strings.TrimSuffix(frontendURL, "/")

	(*w).Header().Set("Access-Control-Allow-Origin", frontendURL)
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
}

// PublicLeaderboardEntry is a safe structure that doesn't expose sensitive data
type PublicLeaderboardEntry struct {
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Score   int    `json:"score"`
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	users, err := db.GetLeaderboardWithMock(10)
	if err != nil {
		log.Printf("[API] Error fetching leaderboard: %v", err)
		http.Error(w, "Failed to fetch leaderboard", http.StatusInternalServerError)
		return
	}

	// Convert to public format - exclude userId and email
	publicEntries := make([]PublicLeaderboardEntry, len(users))
	for i, u := range users {
		publicEntries[i] = PublicLeaderboardEntry{
			Name:    u.Name,
			Picture: u.Picture,
			Score:   u.Score,
		}
	}
	json.NewEncoder(w).Encode(publicEntries)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "Missing userId parameter", http.StatusBadRequest)
		return
	}

	games, err := db.GetGameHistoryWithMock(userID, 20)
	if err != nil {
		log.Printf("[API] Error fetching history: %v", err)
		http.Error(w, "Failed to fetch history", http.StatusInternalServerError)
		return
	}

	// Get total count of games for this player
	totalCount, err := db.CountGamesByPlayerWithMock(userID)
	if err != nil {
		log.Printf("[API] Error counting games: %v", err)
		totalCount = len(games) // Fallback to length of returned games
	}

	response := map[string]interface{}{
		"games":      games,
		"totalCount": totalCount,
	}
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, using environment variables")
	}

	// Initialize OAuth
	initOAuth()

	// Initialize DB (with mock support for local development)
	db.InitWithMocks()

	// Initialize Redis/Valkey for distributed matchmaking
	if err := InitRedis(); err != nil {
		log.Printf("Warning: Redis not available, using in-memory matchmaking (single-pod mode)")
	}

	// Initialize Game Manager
	gameManager := NewGameManager()
	go gameManager.Run()

	// Start matchmaking loop and event subscriptions if Redis is available
	if IsRedisAvailable() {
		go gameManager.RunMatchmakingLoop()
		go gameManager.SubscribeToMatchNotifications()
		go gameManager.SubscribeToGameEvents() // Subscribe to distributed game events
		log.Println("Distributed matchmaking and game events enabled via Redis")
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Register handlers
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api", apiHandler)
	http.HandleFunc("/auth/google/login", handleGoogleLogin)
	http.HandleFunc("/auth/google/callback", handleGoogleCallback)
	http.HandleFunc("/auth/verify", handleVerifySession)
	http.HandleFunc("/auth/logout", handleLogout)
	http.HandleFunc("/api/leaderboard", handleLeaderboard)
	http.HandleFunc("/api/history", handleHistory)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(gameManager, w, r)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := Response{
			Message: "Cookie Clicker Backend",
			Status:  "online",
		}
		json.NewEncoder(w).Encode(response)
	})

	// Start server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
