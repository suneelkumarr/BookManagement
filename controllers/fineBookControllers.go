package controllers

import (
	"database/sql"
	database "go-crud-api/config"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// FineBook represents the structure of a fine record in the FineBookTable
type FineBook struct {
	FineID     int     `json:"FineID"`
	PersonID   int     `json:"PersonID"`
	OrderID    int     `json:"OrderID"`
	FineTypeID int     `json:"FineTypeID"`
	FineAmount float64 `json:"FineAmount"`
}

// CreateFineBook handles the creation of a new fine record
func CreateFineBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		var newFine FineBook
		if err := c.ShouldBindJSON(&newFine); err != nil {
			log.Printf("invalid request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		// Validate input
		if newFine.PersonID <= 0 || newFine.OrderID <= 0 || newFine.FineTypeID <= 0 || newFine.FineAmount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "all fields must be positive"})
			return
		}

		// Insert into database
		insertQuery := `
            INSERT INTO FineBookTable (PersonID, OrderID, FineTypeID, FineAmount)
            VALUES (?, ?, ?, ?);
            SELECT SCOPE_IDENTITY() AS FineID;`
		err := db.QueryRow(insertQuery,
			newFine.PersonID,
			newFine.OrderID,
			newFine.FineTypeID,
			newFine.FineAmount,
		).Scan(&newFine.FineID)
		if err != nil {
			log.Printf("insert fine: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create fine record"})
			return
		}

		c.JSON(http.StatusCreated, newFine)
	}
}

// GetAllFineBooks retrieves all fine records (top 1000)
func GetAllFineBooks() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		query := `SELECT TOP (1000) FineID, PersonID, OrderID, FineTypeID, FineAmount FROM FineBookTable`
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("get all fines: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get fine records"})
			return
		}
		defer rows.Close()

		var fines []FineBook
		for rows.Next() {
			var fine FineBook
			err := rows.Scan(
				&fine.FineID,
				&fine.PersonID,
				&fine.OrderID,
				&fine.FineTypeID,
				&fine.FineAmount,
			)
			if err != nil {
				log.Printf("scan fine: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan fine"})
				return
			}
			fines = append(fines, fine)
		}

		if err := rows.Err(); err != nil {
			log.Printf("get all fines: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get fine records"})
			return
		}

		c.JSON(http.StatusOK, fines)
	}
}

// GetFineBookByID retrieves a specific fine record by ID
func GetFineBookByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		fineID, err := strconv.Atoi(c.Param("id"))
		if err != nil || fineID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fine_id"})
			return
		}

		query := `SELECT FineID, PersonID, OrderID, FineTypeID, FineAmount FROM FineBookTable WHERE FineID = ?`
		var fine FineBook
		err = db.QueryRow(query, fineID).Scan(
			&fine.FineID,
			&fine.PersonID,
			&fine.OrderID,
			&fine.FineTypeID,
			&fine.FineAmount,
		)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "fine record not found"})
			return
		}
		if err != nil {
			log.Printf("query fine %d: %v", fineID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve fine record"})
			return
		}

		c.JSON(http.StatusOK, fine)
	}
}

// UpdateFineBook handles updating an existing fine record
func UpdateFineBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		fineID, err := strconv.Atoi(c.Param("id"))
		if err != nil || fineID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fine_id"})
			return
		}

		var updateFine FineBook
		if err := c.ShouldBindJSON(&updateFine); err != nil {
			log.Printf("invalid request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		// Validate input
		if updateFine.PersonID <= 0 || updateFine.OrderID <= 0 || updateFine.FineTypeID <= 0 || updateFine.FineAmount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "all fields must be positive"})
			return
		}

		// Update in database
		updateQuery := `
            UPDATE FineBookTable
            SET PersonID = ?, OrderID = ?, FineTypeID = ?, FineAmount = ?
            WHERE FineID = ?`
		result, err := db.Exec(updateQuery,
			updateFine.PersonID,
			updateFine.OrderID,
			updateFine.FineTypeID,
			updateFine.FineAmount,
			fineID,
		)
		if err != nil {
			log.Printf("update fine %d: %v", fineID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update fine record"})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("check rows affected: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify update"})
			return
		}
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "fine record not found"})
			return
		}

		updateFine.FineID = fineID
		c.JSON(http.StatusOK, updateFine)
	}
}
