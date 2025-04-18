// helper/jwt_helper.go
package helper

import (
	"context"
	"database/sql"
	"fmt"
	database "go-crud-api/config"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// -----------------------------------------------------------------------------
// Models & globals
// -----------------------------------------------------------------------------

// SignedDetails are the custom claims we embed in every JWT.
type SignedDetails struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Uid       string `json:"uid"`
	jwt.RegisteredClaims
}

var (
	// SECRET_KEY is loaded from the environment when the package initialises.
	SECRET_KEY string

	// db is injected via SetDB and reused by helper functions that need DB access.
	db *sql.DB
)

// -----------------------------------------------------------------------------
// Package initialisation
// -----------------------------------------------------------------------------

func init() {
	SECRET_KEY = "todoapp"
	if SECRET_KEY == "" {
		log.Fatal("SECRET_KEY environment variable is not set")
	}
}

// -----------------------------------------------------------------------------
// Public helpers
// -----------------------------------------------------------------------------

// SetDB injects a *sql.DB connection for later use (e.g., UpdateAllTokens).
func SetDB(conn *sql.DB) { db = conn }

// HashPassword hashes the given plain‑text password using bcrypt.
func HashPassword(pw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// GenerateAllTokens returns an access token (24 h) and a refresh token (7 d).
func GenerateAllTokens(email, first, last, uid string) (accessToken, refreshToken string, err error) {
	now := time.Now()

	accessClaims := &SignedDetails{
		Email:     email,
		FirstName: first,
		LastName:  last,
		Uid:       uid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	refreshClaims := &SignedDetails{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	accessToken, err = jwt.
		NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err = jwt.
		NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// UpdateAllTokens stores the freshly generated tokens in the Person table.
// UpdateAllTokens updates the Token and Refresh_Token for a user
func UpdateAllTokens(access, refresh, userID string) error {
	db := database.Database()
	if db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use positional parameters for simplicity
	const stmt = `
        UPDATE Person
        SET Token = ?,
            Refresh_Token = ?,
            Updated_at = ?
        WHERE User_id = ?`

	// Execute the update
	result, err := db.ExecContext(ctx, stmt,
		access,
		refresh,
		time.Now(),
		userID,
	)
	if err != nil {
		log.Printf("failed to update tokens for user %s: %v", userID, err)
		return fmt.Errorf("failed to update tokens for user %s: %w", userID, err)
	}

	// Check rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("failed to check rows affected for user %s: %v", userID, err)
		return fmt.Errorf("failed to verify update for user %s: %w", userID, err)
	}
	if rowsAffected == 0 {
		log.Printf("no user found with User_id %s", userID)
		return fmt.Errorf("no user found with User_id %s", userID)
	}

	return nil
}

// ValidateToken parses and validates a JWT, returning its claims or an error message.
func ValidateToken(raw string) (*SignedDetails, string) {
	token, err := jwt.ParseWithClaims(
		raw,
		&SignedDetails{},
		func(t *jwt.Token) (any, error) { return []byte(SECRET_KEY), nil },
	)
	if err != nil {
		return nil, fmt.Sprintf("invalid token: %v", err)
	}

	claims, ok := token.Claims.(*SignedDetails)
	if !ok || !token.Valid {
		return nil, "token is invalid"
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, "token is expired"
	}
	return claims, ""
}

// GenerateUUID returns a random v4 UUID as a string.
func GenerateUUID() string { return uuid.NewString() }

func VerifyPassword(userPassword string, providedPassword string) (bool, string) {

	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := true
	msg := ""

	if err != nil {
		msg = fmt.Sprintf("login or password is incorrect")
		check = false
	}
	return check, msg
}
