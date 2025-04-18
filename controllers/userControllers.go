package controllers

import (
	"database/sql"
	database "go-crud-api/config"
	"go-crud-api/helper"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PhoneNumber  string    `json:"phonenumber"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Password     string    `json:"Password"`
	Token        string    `json:"token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	UserID       string    `json:"user_id"`
}

func CreateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()

		var newUser User
		if err := c.ShouldBindJSON(&newUser); err != nil {
			log.Printf("invalid request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// basic sanitation
		newUser.Username = strings.TrimSpace(newUser.Username)
		newUser.Email = strings.TrimSpace(newUser.Email)
		newUser.Password = strings.TrimSpace(newUser.Password)

		if newUser.Username == "" || newUser.Email == "" || newUser.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username, email and password are required"})
			return
		}
		if !strings.Contains(newUser.Email, "@") || !strings.Contains(newUser.Email, ".") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
			return
		}

		// check duplicates
		const dupChk = `SELECT 1 FROM Person WHERE Username = ? OR Email = ?`
		var dummy int
		err := db.QueryRow(dupChk,
			newUser.Username,
			newUser.Email,
		).Scan(&dummy)
		switch {
		case err == nil:
			c.JSON(http.StatusConflict, gin.H{"error": "username or email already exists"})
			return
		case err != sql.ErrNoRows:
			log.Printf("dup-check error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		// hash password
		hashed, err := helper.HashPassword(newUser.Password)
		if err != nil {
			log.Printf("hash password: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
			return
		}
		newUser.Password = hashed // save hashed password for later
		// timestamps and ids
		now := time.Now()
		newUser.CreatedAt = now
		newUser.UpdatedAt = now
		newUser.UserID = helper.GenerateUUID()

		// issue JWTs
		access, refresh, err := helper.GenerateAllTokens(
			newUser.Email, newUser.FirstName, newUser.LastName, newUser.UserID)
		if err != nil {
			log.Printf("generate tokens: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
			return
		}

		newUser.Token, newUser.RefreshToken = access, refresh

		// insert
		const ins = `
			INSERT INTO Person
        (Username, Email, First_name, Last_name, Password, PhoneNumber,
         Created_at, Updated_at, User_id, Token, Refresh_Token)
        VALUES (?,?,?,?,?,?,?,?,?,?,?);
    SELECT SCOPE_IDENTITY() AS ID;
		`

		err = db.QueryRow(ins,
			newUser.Username,
			newUser.Email,
			newUser.FirstName,
			newUser.LastName,
			hashed,
			newUser.PhoneNumber,
			newUser.CreatedAt,
			newUser.UpdatedAt,
			newUser.UserID,
			newUser.Token,
			newUser.RefreshToken,
		).Scan(&newUser.ID)
		if err != nil {
			log.Printf("insert user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		c.JSON(http.StatusCreated, newUser)
	}
}

// GetUsers retrieves all users from the Person table
func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()

		const q = `
            SELECT ID, Username, Email, PhoneNumber, First_name, Last_name,
                   Created_at, Updated_at, User_id
            FROM Person`
		rows, err := db.Query(q)
		if err != nil {
			log.Printf("list users: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve users"})
			return
		}
		defer rows.Close()

		var users []User
		for rows.Next() {
			var u User
			var phoneNumber, firstName, lastName sql.NullString // Handle nullable fields

			if err := rows.Scan(
				&u.ID,
				&u.Username,
				&u.Email,
				&phoneNumber,
				&firstName,
				&lastName,
				&u.CreatedAt,
				&u.UpdatedAt,
				&u.UserID,
			); err != nil {
				log.Printf("scan users: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process users"})
				return
			}

			// Convert nullable fields to strings
			u.PhoneNumber = phoneNumber.String
			u.FirstName = firstName.String
			u.LastName = lastName.String

			users = append(users, u)
		}

		// Check for errors from iterating over rows
		if err := rows.Err(); err != nil {
			log.Printf("rows error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process users"})
			return
		}

		c.JSON(http.StatusOK, users)
	}
}

// GetUserById retrieves a user by their UserID
func GetUserById() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		uid := c.Param("user_id")

		// Ensure database connection is valid
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		const q = `
            SELECT ID, Username, Email, PhoneNumber, First_name, Last_name,
                   Created_at, Updated_at, User_id
            FROM Person
            WHERE User_id = ?`
		row := db.QueryRow(q, uid)

		var u User
		var phoneNumber, firstName, lastName sql.NullString
		if err := row.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&phoneNumber,
			&firstName,
			&lastName,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.UserID,
		); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			log.Printf("get user by id %s: %v", uid, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
			return
		}

		// Convert nullable fields to strings
		u.PhoneNumber = phoneNumber.String
		u.FirstName = firstName.String
		u.LastName = lastName.String

		c.JSON(http.StatusOK, u)
	}
}

// -----------------------------------------------------------------------------
// GET /users/name/:username  — single user by username
// -----------------------------------------------------------------------------

func GetUserByName() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		username := c.Param("username")

		const q = `
			SELECT ID, Username, Email, PhoneNumber, FirstName, LastName,
			       CreatedAt, UpdatedAt, UserID
			  FROM Person
			 WHERE Username = @p1`
		row := db.QueryRow(q, sql.Named("p1", username))

		var u User
		var PhoneNumber, first, last sql.NullString
		if err := row.Scan(&u.ID, &u.Username, &u.Email, &PhoneNumber,
			&first, &last, &u.CreatedAt, &u.UpdatedAt, &u.UserID); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			log.Printf("get user by name: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		u.PhoneNumber, u.FirstName, u.LastName = PhoneNumber.String, first.String, last.String
		c.JSON(http.StatusOK, u)
	}
}

// UpdateUserById dynamically updates a user by their User_id
func UpdateUserById() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		uid := c.Param("user_id")

		// Ensure database connection is valid
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		// Bind JSON payload to a map for dynamic updates
		var payload map[string]interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			log.Printf("invalid request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// Validate user_id
		if uid == "" || len(uid) > 36 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}

		// Fields allowed to be updated
		allowedFields := map[string]string{
			"username":     "Username",
			"email":        "Email",
			"phone_number": "PhoneNumber",
			"first_name":   "First_name",
			"last_name":    "Last_name",
		}

		// Build dynamic SQL query
		var setClauses []string
		var args []interface{}
		for jsonKey, dbColumn := range allowedFields {
			if value, exists := payload[jsonKey]; exists {
				// Basic validation for non-empty strings
				if str, ok := value.(string); ok {
					str = strings.TrimSpace(str)
					if str == "" && jsonKey != "phone_number" && jsonKey != "first_name" && jsonKey != "last_name" {
						continue // Skip empty strings for non-nullable fields
					}
					if jsonKey == "email" && (!strings.Contains(str, "@") || !strings.Contains(str, ".")) {
						c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
						return
					}
					setClauses = append(setClauses, dbColumn+" = ?")
					args = append(args, str)
				}
			}
		}

		// Always update Updated_at
		setClauses = append(setClauses, "Updated_at = ?")
		args = append(args, time.Now())

		// If no fields to update, return error
		if len(setClauses) == 1 { // Only Updated_at
			c.JSON(http.StatusBadRequest, gin.H{"error": "no valid fields provided to update"})
			return
		}

		// Construct the UPDATE query
		query := "UPDATE Person SET " + strings.Join(setClauses, ", ") + " WHERE User_id = ?"
		args = append(args, uid)

		// Execute the update
		result, err := db.Exec(query, args...)
		if err != nil {
			log.Printf("update user %s: %v", uid, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}

		// Check if any rows were affected
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("check rows affected: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify update"})
			return
		}
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Retrieve the updated user
		const selectQuery = `
            SELECT ID, Username, Email, PhoneNumber, First_name, Last_name,
                   Created_at, Updated_at, User_id
            FROM Person
            WHERE User_id = ?`
		row := db.QueryRow(selectQuery, uid)

		var u User
		var phoneNumber, firstName, lastName sql.NullString
		if err := row.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&phoneNumber,
			&firstName,
			&lastName,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.UserID,
		); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			log.Printf("get updated user %s: %v", uid, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve updated user"})
			return
		}

		// Convert nullable fields to strings
		u.PhoneNumber = phoneNumber.String
		u.FirstName = firstName.String
		u.LastName = lastName.String

		c.JSON(http.StatusOK, u)
	}
}

// LoginUser authenticates a user and generates tokens
func LoginUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		// Bind JSON payload
		var input struct {
			Username string `json:"username"`
			Password string `json:"Password"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			log.Printf("invalid request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		// Validate input
		input.Username = strings.TrimSpace(input.Username)
		input.Password = strings.TrimSpace(input.Password)
		if input.Username == "" || input.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
			return
		}

		// Query user by username
		const query = `
            SELECT ID, Username, Email, PhoneNumber, First_name, Last_name, 
                   Password, Created_at, Updated_at, User_id
            FROM Person 
            WHERE Username = ?`
		row := db.QueryRow(query, input.Username)

		var user User
		var phoneNumber, firstName, lastName sql.NullString
		var hashedPassword string
		if err := row.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&phoneNumber,
			&firstName,
			&lastName,
			&hashedPassword,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.UserID,
		); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
				return
			}
			log.Printf("get user %s: %v", input.Username, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
			return
		}

		// Convert nullable fields
		user.PhoneNumber = phoneNumber.String
		user.FirstName = firstName.String
		user.LastName = lastName.String

		// Verify password
		passwordIsValid, msg := helper.VerifyPassword(input.Password, hashedPassword)
		if !passwordIsValid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
		}

		// Generate tokens
		token, refreshToken, err := helper.GenerateAllTokens(
			user.Email,
			user.FirstName,
			user.LastName,
			user.UserID,
		)
		if err != nil {
			log.Printf("generate tokens for %s: %v", user.Username, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
			return
		}

		// Update tokens in the database
		err = helper.UpdateAllTokens(token, refreshToken, user.UserID)
		if err != nil {
			log.Printf("update tokens for %s: %v", user.Username, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update tokens"})
			return
		}

		// Prepare response
		response := struct {
			User         User   `json:"user"`
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		}{
			User:         user,
			Token:        token,
			RefreshToken: refreshToken,
		}

		c.JSON(http.StatusOK, response)
	}
}
