package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOauthConfig *oauth2.Config
	oauthStateString  string
	jwtSecret         []byte
)

type UserSession struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Token   string `json:"token"`
}

type JWTClaims struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	jwt.RegisteredClaims
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

func initOAuth() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	// Validate required OAuth credentials
	if clientID == "" || clientID == "your_google_client_id_here" {
		log.Fatal("ERROR: GOOGLE_CLIENT_ID is not set or is still using placeholder value. Please set it in your .env file.")
	}
	if clientSecret == "" || clientSecret == "your_google_client_secret_here" {
		log.Fatal("ERROR: GOOGLE_CLIENT_SECRET is not set or is still using placeholder value. Please set it in your .env file.")
	}

	if redirectURL == "" {
		redirectURL = "http://localhost:8080/auth/google/callback"
	}

	log.Printf("Initializing OAuth with Client ID: %s...", clientID[:10])

	googleOauthConfig = &oauth2.Config{
		RedirectURL:  redirectURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// Generate random state string
	b := make([]byte, 32)
	rand.Read(b)
	oauthStateString = base64.URLEncoding.EncodeToString(b)

	// Initialize JWT secret
	jwtSecretStr := os.Getenv("JWT_SECRET")
	if jwtSecretStr == "" {
		// Generate a random secret if not provided
		secret := make([]byte, 32)
		rand.Read(secret)
		jwtSecret = secret
		log.Println("Warning: JWT_SECRET not set, using randomly generated secret")
	} else {
		jwtSecret = []byte(jwtSecretStr)
	}
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	log.Printf("[AUTH] Google OAuth login initiated from IP: %s", r.RemoteAddr)
	url := googleOauthConfig.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
	log.Printf("[AUTH] Redirecting to Google OAuth URL")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("[AUTH] OAuth callback received from IP: %s", r.RemoteAddr)
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	state := r.FormValue("state")
	if state != oauthStateString {
		log.Printf("[AUTH] ERROR: Invalid OAuth state - potential CSRF attack from IP: %s", r.RemoteAddr)
		http.Redirect(w, r, fmt.Sprintf("%s/login?error=invalid_state", frontendURL), http.StatusTemporaryRedirect)
		return
	}
	log.Printf("[AUTH] OAuth state validated successfully")

	code := r.FormValue("code")
	log.Printf("[AUTH] Exchanging authorization code for token")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("[AUTH] ERROR: Code exchange failed: %s", err.Error())
		http.Redirect(w, r, fmt.Sprintf("%s/login?error=exchange_failed", frontendURL), http.StatusTemporaryRedirect)
		return
	}
	log.Printf("[AUTH] Successfully exchanged code for access token")

	// Get user info from Google
	log.Printf("[AUTH] Fetching user info from Google")
	userInfo, err := getUserInfo(token.AccessToken)
	if err != nil {
		log.Printf("[AUTH] ERROR: Failed to get user info: %s", err.Error())
		http.Redirect(w, r, fmt.Sprintf("%s/login?error=userinfo_failed", frontendURL), http.StatusTemporaryRedirect)
		return
	}
	log.Printf("[AUTH] User info retrieved: Email=%s, Name=%s, ID=%s", userInfo.Email, userInfo.Name, userInfo.ID)

	// Create JWT token
	log.Printf("[AUTH] Generating JWT token for user: %s", userInfo.Email)
	jwtToken, err := generateJWT(userInfo)
	if err != nil {
		log.Printf("[AUTH] ERROR: Failed to generate JWT: %s", err.Error())
		http.Redirect(w, r, fmt.Sprintf("%s/login?error=token_generation_failed", frontendURL), http.StatusTemporaryRedirect)
		return
	}
	log.Printf("[AUTH] JWT token generated successfully for user: %s", userInfo.Email)

	// Redirect to frontend with JWT token
	log.Printf("[AUTH] Redirecting user %s to frontend callback", userInfo.Email)
	http.Redirect(w, r, fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, jwtToken), http.StatusTemporaryRedirect)
}

func getUserInfo(accessToken string) (*GoogleUserInfo, error) {
	log.Printf("[AUTH] Requesting user info from Google API")
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		log.Printf("[AUTH] ERROR: HTTP request to Google API failed: %s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[AUTH] ERROR: Google API returned status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[AUTH] ERROR: Failed to read response body: %s", err.Error())
		return nil, err
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(data, &userInfo); err != nil {
		log.Printf("[AUTH] ERROR: Failed to unmarshal user info: %s", err.Error())
		return nil, err
	}

	log.Printf("[AUTH] Successfully retrieved user info from Google")
	return &userInfo, nil
}

func generateJWT(userInfo *GoogleUserInfo) (string, error) {
	log.Printf("[AUTH] Creating JWT claims for user: %s", userInfo.Email)
	claims := JWTClaims{
		UserID:  userInfo.ID,
		Email:   userInfo.Email,
		Name:    userInfo.Name,
		Picture: userInfo.Picture,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "overcookied",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("[AUTH] ERROR: Failed to sign JWT token: %s", err.Error())
		return "", err
	}
	log.Printf("[AUTH] JWT token signed successfully, expires in 24 hours")
	return signedToken, nil
}

func verifyJWT(tokenString string) (*JWTClaims, error) {
	log.Printf("[AUTH] Verifying JWT token")
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("[AUTH] ERROR: Unexpected signing method: %v", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		log.Printf("[AUTH] ERROR: JWT parsing failed: %s", err.Error())
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		log.Printf("[AUTH] JWT token verified successfully for user: %s", claims.Email)
		return claims, nil
	}

	log.Printf("[AUTH] ERROR: Invalid JWT token")
	return nil, fmt.Errorf("invalid token")
}

func handleVerifySession(w http.ResponseWriter, r *http.Request) {
	log.Printf("[AUTH] Session verification request from IP: %s", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")

	// Enable CORS
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	w.Header().Set("Access-Control-Allow-Origin", frontendURL)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		log.Printf("[AUTH] CORS preflight request handled")
		w.WriteHeader(http.StatusOK)
		return
	}

	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		log.Printf("[AUTH] ERROR: No token provided in request from IP: %s", r.RemoteAddr)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "no token provided"})
		return
	}

	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Verify JWT
	claims, err := verifyJWT(tokenString)
	if err != nil {
		log.Printf("[AUTH] ERROR: Session verification failed from IP: %s - %s", r.RemoteAddr, err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired token"})
		return
	}

	log.Printf("[AUTH] Session verified successfully for user: %s", claims.Email)

	// Return user session info
	session := &UserSession{
		ID:      claims.UserID,
		Email:   claims.Email,
		Name:    claims.Name,
		Picture: claims.Picture,
		Token:   tokenString,
	}

	json.NewEncoder(w).Encode(session)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	log.Printf("[AUTH] Logout request from IP: %s", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")

	// Enable CORS
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	w.Header().Set("Access-Control-Allow-Origin", frontendURL)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		log.Printf("[AUTH] CORS preflight request handled")
		w.WriteHeader(http.StatusOK)
		return
	}

	// With JWT, logout is handled client-side by removing the token
	// Server doesn't need to maintain state
	log.Printf("[AUTH] User logged out successfully (client-side token removal)")
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out successfully"})
}
