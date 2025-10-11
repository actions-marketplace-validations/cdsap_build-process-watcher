package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cdsap/build-process-watcher/backend/internal/models"
)

var (
	secretKey   string
	adminSecret string
)

// Initialize loads secrets from environment variables
func Initialize() {
	secretKey = getSecretKey()
	adminSecret = getAdminSecret()
}

// getSecretKey returns the secret key from environment variable or a default for development
func getSecretKey() string {
	key := os.Getenv("JWT_SECRET_KEY")
	if key == "" {
		// Use a default key for development/testing only
		// In production, this should always be set via environment variable
		return "build-process-watcher-secret-2024-dev"
	}
	return key
}

// getAdminSecret returns the admin secret from environment variable or a default for development
func getAdminSecret() string {
	secret := os.Getenv("ADMIN_SECRET")
	if secret == "" {
		// Use a default secret for development/testing only
		// In production, this MUST be set via environment variable
		log.Printf("⚠️  WARNING: ADMIN_SECRET not set, using default (insecure for production!)")
		return "admin-dev-secret-change-me"
	}
	return secret
}

// RequireAdminAuth checks if the request has valid admin authentication
func RequireAdminAuth(r *http.Request) bool {
	// Check for admin secret in header
	providedSecret := r.Header.Get("X-Admin-Secret")
	if providedSecret == "" {
		return false
	}
	return providedSecret == adminSecret
}

// SetAdminSecretForTest allows tests to override the admin secret (test use only!)
func SetAdminSecretForTest(secret string) {
	adminSecret = secret
}

// GetAdminSecret returns the current admin secret (test use only!)
func GetAdminSecret() string {
	return adminSecret
}

// GenerateToken generates a JWT token for a specific run
func GenerateToken(runID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(2 * time.Hour) // Token expires in 2 hours
	
	tokenData := models.TokenData{
		RunID:     runID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	
	// Encode token data as JSON
	payload, err := json.Marshal(tokenData)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal token data: %w", err)
	}
	
	// Create HMAC signature
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(payload)
	signature := mac.Sum(nil)
	
	// Combine payload and signature
	token := base64.URLEncoding.EncodeToString(payload) + "." + hex.EncodeToString(signature)
	
	return token, expiresAt, nil
}

// ValidateToken validates a JWT token for a specific run
func ValidateToken(token string, runID string) (bool, error) {
	// Split token into payload and signature
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid token format")
	}
	
	payloadEncoded := parts[0]
	signatureHex := parts[1]
	
	// Decode payload
	payload, err := base64.URLEncoding.DecodeString(payloadEncoded)
	if err != nil {
		return false, fmt.Errorf("failed to decode payload: %w", err)
	}
	
	// Decode signature
	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}
	
	// Verify signature
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(payload)
	expectedSignature := mac.Sum(nil)
	
	if !hmac.Equal(signature, expectedSignature) {
		return false, fmt.Errorf("invalid signature")
	}
	
	// Parse token data
	var tokenData models.TokenData
	if err := json.Unmarshal(payload, &tokenData); err != nil {
		return false, fmt.Errorf("failed to unmarshal token data: %w", err)
	}
	
	// Check if token has expired
	if time.Now().After(tokenData.ExpiresAt) {
		return false, fmt.Errorf("token has expired")
	}
	
	// Check if token is for the correct run_id
	if tokenData.RunID != runID {
		return false, fmt.Errorf("token run_id mismatch")
	}
	
	return true, nil
}

