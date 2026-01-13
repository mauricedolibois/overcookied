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

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/mauricedolibois/overcookied/backend/db"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOauthConfig *oauth2.Config
	jwtSecret         []byte
)

// generateOAuthState creates a unique state string for CSRF protection
func generateOAuthState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

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

// OAuthSecrets represents the structure of OAuth credentials stored in AWS Secrets Manager
type OAuthSecrets struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// fetchOAuthFromSecretsManager fetches OAuth credentials from AWS Secrets Manager
func fetchOAuthFromSecretsManager() (clientID, clientSecret string, err error) {
	secretName := os.Getenv("GOOGLE_OAUTH_SECRET_NAME")
	if secretName == "" {
		return "", "", fmt.Errorf("GOOGLE_OAUTH_SECRET_NAME not set")
	}

	log.Printf("[AUTH] Fetching OAuth credentials from AWS Secrets Manager: %s", secretName)

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return "", "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := secretsmanager.NewFromConfig(cfg)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	}

	result, err := client.GetSecretValue(context.Background(), input)
	if err != nil {
		return "", "", fmt.Errorf("failed to get secret: %w", err)
	}

	var secrets OAuthSecrets
	if err := json.Unmarshal([]byte(*result.SecretString), &secrets); err != nil {
		return "", "", fmt.Errorf("failed to parse secret: %w", err)
	}

	log.Printf("[AUTH] Successfully fetched OAuth credentials from Secrets Manager")
	return secrets.ClientID, secrets.ClientSecret, nil
}

func initOAuth() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	// If credentials not in env vars, try AWS Secrets Manager
	if clientID == "" || clientID == "your_google_client_id_here" || clientSecret == "" || clientSecret == "your_google_client_secret_here" {
		log.Println("[AUTH] OAuth credentials not found in environment variables, trying AWS Secrets Manager...")
		var err error
		clientID, clientSecret, err = fetchOAuthFromSecretsManager()
		if err != nil {
			log.Fatalf("ERROR: Failed to fetch OAuth credentials from Secrets Manager: %v. Please set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET environment variables or configure AWS Secrets Manager.", err)
		}
	}

	// Validate required OAuth credentials
	if clientID == "" {
		log.Fatal("ERROR: GOOGLE_CLIENT_ID is not set or is still using placeholder value.")
	}
	if clientSecret == "" {
		log.Fatal("ERROR: GOOGLE_CLIENT_SECRET is not set or is still using placeholder value.")
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

	// Generate a unique state for this request
	state := generateOAuthState()

	// Store state in a cookie (works across multiple replicas)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
		Secure:   strings.HasPrefix(os.Getenv("GOOGLE_REDIRECT_URL"), "https://"),
		SameSite: http.SameSiteLaxMode,
	})

	url := googleOauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	log.Printf("[AUTH] Redirecting to Google OAuth URL")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("[AUTH] OAuth callback received from IP: %s", r.RemoteAddr)
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	// Get the expected state from the cookie
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		log.Printf("[AUTH] ERROR: OAuth state cookie not found: %v", err)
		http.Redirect(w, r, fmt.Sprintf("%s/login?error=invalid_state", frontendURL), http.StatusTemporaryRedirect)
		return
	}
	expectedState := stateCookie.Value

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	state := r.FormValue("state")
	if state != expectedState {
		log.Printf("[AUTH] ERROR: Invalid OAuth state - potential CSRF attack from IP: %s (expected: %s, got: %s)", r.RemoteAddr, expectedState[:10], state[:10])
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

	// PERSIST USER in DynamoDB
	user := db.CookieUser{
		UserID:  userInfo.ID,
		Email:   userInfo.Email,
		Name:    userInfo.Name,
		Picture: userInfo.Picture,
	}
	if err := db.SaveUser(user); err != nil {
		log.Printf("[AUTH] WARNING: Failed to save user to DB: %v", err)
	} else {
		log.Printf("[AUTH] User saved to DynamoDB: %s", userInfo.Email)
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
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Upgrade, Connection")

	if r.Method == "OPTIONS" {
		log.Printf("[AUTH] CORS preflight request handled for Origin: %s", origin)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("[AUTH] Processing verification request method: %s", r.Method)

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
