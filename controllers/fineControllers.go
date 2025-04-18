package controllers

import (
	"database/sql"
	database "go-crud-api/config"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Fine struct {
	FineID     int
	NameOfFine string
	FineAmount float64
}

// CreateFine handles the creation of a new fine
func CreateFine() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database() // Assumes database.Database() returns *sql.DB
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		// Parse the request body into a Fine struct
		var newFine Fine
		if err := c.ShouldBindJSON(&newFine); err != nil {
			log.Printf("invalid request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		// Sanitize and validate input
		newFine.NameOfFine = strings.TrimSpace(newFine.NameOfFine)
		if newFine.NameOfFine == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name_of_fine is required"})
			return
		}
		if len(newFine.NameOfFine) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name_of_fine is too long (max 100 characters)"})
			return
		}
		if newFine.FineAmount < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fine_amount must be non-negative"})
			return
		}
		if newFine.FineAmount > 1000000 { // Example upper limit to prevent overflow
			c.JSON(http.StatusBadRequest, gin.H{"error": "fine_amount is too large (max 1000000)"})
			return
		}

		// Check if a fine with the same name already exists
		var existingID int
		checkQuery := "SELECT FineID FROM FineTable WHERE NameOfFine = ?"
		err := db.QueryRow(checkQuery, newFine.NameOfFine).Scan(&existingID)
		if err == nil {
			log.Printf("fine with name %s already exists, ID: %d", newFine.NameOfFine, existingID)
			c.JSON(http.StatusConflict, gin.H{
				"error":      "a fine with this name already exists",
				"existingID": existingID,
			})
			return
		} else if err != sql.ErrNoRows {
			log.Printf("check fine existence for %s: %v", newFine.NameOfFine, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check fine existence"})
			return
		}

		// Insert the new fine into the database
		insertQuery := `
            INSERT INTO FineTable (NameOfFine, FineAmount)
            VALUES (?, ?);
            SELECT SCOPE_IDENTITY() AS FineID;`
		err = db.QueryRow(insertQuery, newFine.NameOfFine, newFine.FineAmount).Scan(&newFine.FineID)
		if err != nil {
			log.Printf("insert fine %s: %v", newFine.NameOfFine, err)
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				c.JSON(http.StatusConflict, gin.H{"error": "a fine with this name already exists"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create fine"})
			}
			return
		}

		// Return the created fine
		c.JSON(http.StatusCreated, newFine)
	}
}

func GetFines() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()

		// Fetch all fines from the database
		query := "SELECT FineID, NameOfFine, FineAmount FROM FineTable"
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("fetch fines: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fines"})
			return
		}
		defer rows.Close()

		// Create a slice to hold the fetched fines
		var fines []Fine

		// Iterate over the rows and populate the fines slice
		for rows.Next() {
			var fine Fine
			err := rows.Scan(&fine.FineID, &fine.NameOfFine, &fine.FineAmount)
			if err != nil {
				log.Printf("scan fine: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan fine"})
				return
			}
			fines = append(fines, fine)
		}

		// Return the fetched fines
		c.JSON(http.StatusOK, fines)
	}
}

// func GetFineById() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		db := database.Database()
// 	}
// }

// func UpdateFineById() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		db := database.Database()
// 	}
// }
