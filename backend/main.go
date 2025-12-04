package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

func main() {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Register handlers
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api", apiHandler)
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
