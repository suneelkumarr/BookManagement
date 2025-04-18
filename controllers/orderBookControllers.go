package controllers

import (
	"database/sql"
	database "go-crud-api/config"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// OrderBook represents the structure of an order in the OrderBook table
type OrderBook struct {
	OrderID          int     `json:"OrderID"`
	PersonID         int     `json:"PersonID"`
	BookID           int     `json:"BookID"`
	BorrowDate       string  `json:"BorrowDate"`       // String in YYYY-MM-DD format
	ReturnDate       *string `json:"ReturnDate"`       // Nullable
	ActualReturnDate *string `json:"ActualReturnDate"` // Nullable
	Status           string  `json:"Status"`
}

// CreateOrderBook handles the creation of a new order
func CreateOrderBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		var newOrder OrderBook
		if err := c.ShouldBindJSON(&newOrder); err != nil {
			log.Printf("invalid request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		// Validate input
		if newOrder.PersonID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "person_id must be positive"})
			return
		}
		if newOrder.BookID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "book_id must be positive"})
			return
		}
		if newOrder.BorrowDate == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "borrow_date is required"})
			return
		}
		borrowDate, err := time.Parse("2006-01-02", newOrder.BorrowDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "borrow_date must be in YYYY-MM-DD format"})
			return
		}
		if newOrder.ReturnDate != nil {
			if _, err := time.Parse("2006-01-02", *newOrder.ReturnDate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "return_date must be in YYYY-MM-DD format"})
				return
			}
		}
		if newOrder.ActualReturnDate != nil {
			if _, err := time.Parse("2006-01-02", *newOrder.ActualReturnDate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "actual_return_date must be in YYYY-MM-DD format"})
				return
			}
		}
		newOrder.Status = strings.TrimSpace(newOrder.Status)
		if newOrder.Status == "" {
			newOrder.Status = "Borrowed" // Default status
		}
		if !isValidStatus(newOrder.Status) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status must be one of: Borrowed, Returned, Overdue"})
			return
		}

		// Insert into database
		insertQuery := `
            INSERT INTO OrderBook (PersonID, BookID, BorrowDate, ReturnDate, ActualReturnDate, Status)
            VALUES (?, ?, ?, ?, ?, ?);
            SELECT SCOPE_IDENTITY() AS OrderID;`
		err = db.QueryRow(insertQuery,
			newOrder.PersonID,
			newOrder.BookID,
			borrowDate,
			newOrder.ReturnDate,
			newOrder.ActualReturnDate,
			newOrder.Status,
		).Scan(&newOrder.OrderID)
		if err != nil {
			log.Printf("insert order: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
			return
		}

		c.JSON(http.StatusCreated, newOrder)
	}
}

// UpdateOrderBook handles updating an existing order
func UpdateOrderBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		orderID, err := strconv.Atoi(c.Param("id"))
		if err != nil || orderID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order_id"})
			return
		}

		var updateOrder OrderBook
		if err := c.ShouldBindJSON(&updateOrder); err != nil {
			log.Printf("invalid request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		// Build dynamic query
		var setClauses []string
		var args []interface{}

		// Validate and add fields to query if provided
		if updateOrder.PersonID > 0 {
			setClauses = append(setClauses, "PersonID = ?")
			args = append(args, updateOrder.PersonID)
		} else if updateOrder.PersonID < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "person_id must be positive"})
			return
		}

		if updateOrder.BookID > 0 {
			setClauses = append(setClauses, "BookID = ?")
			args = append(args, updateOrder.BookID)
		} else if updateOrder.BookID < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "book_id must be positive"})
			return
		}

		if updateOrder.BorrowDate != "" {
			borrowDate, err := time.Parse("2006-01-02", updateOrder.BorrowDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "borrow_date must be in YYYY-MM-DD format"})
				return
			}
			setClauses = append(setClauses, "BorrowDate = ?")
			args = append(args, borrowDate)
		}

		if updateOrder.ReturnDate != nil {
			if _, err := time.Parse("2006-01-02", *updateOrder.ReturnDate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "return_date must be in YYYY-MM-DD format"})
				return
			}
			setClauses = append(setClauses, "ReturnDate = ?")
			args = append(args, *updateOrder.ReturnDate)
		}

		if updateOrder.ActualReturnDate != nil {
			if _, err := time.Parse("2006-01-02", *updateOrder.ActualReturnDate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "actual_return_date must be in YYYY-MM-DD format"})
				return
			}
			setClauses = append(setClauses, "ActualReturnDate = ?")
			args = append(args, *updateOrder.ActualReturnDate)
		}

		if updateOrder.Status != "" {
			updateOrder.Status = strings.TrimSpace(updateOrder.Status)
			if !isValidStatus(updateOrder.Status) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "status must be one of: Borrowed, Returned, Overdue"})
				return
			}
			setClauses = append(setClauses, "Status = ?")
			args = append(args, updateOrder.Status)
		}

		// Check if there are fields to update
		if len(setClauses) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no valid fields provided for update"})
			return
		}

		// Construct the query
		updateQuery := "UPDATE OrderBook SET " + strings.Join(setClauses, ", ") + " WHERE OrderID = ?"
		args = append(args, orderID)

		// Execute the query
		result, err := db.Exec(updateQuery, args...)
		if err != nil {
			log.Printf("update order %d: %v", orderID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order"})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("check rows affected: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify update"})
			return
		}
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}

		updateOrder.OrderID = orderID
		c.JSON(http.StatusOK, updateOrder)
	}
}

// GetAllOrderBooks retrieves all orders
func GetAllOrderBooks() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		query := `
			SELECT OrderID, PersonID, BookID, BorrowDate, ReturnDate, ActualReturnDate, Status
			FROM OrderBook`
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("get all orders: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get orders"})
			return
		}
		defer rows.Close()

		var orders []OrderBook
		for rows.Next() {
			var order OrderBook
			var borrowDate time.Time
			var returnDate, actualReturnDate sql.NullTime

			err := rows.Scan(
				&order.OrderID,
				&order.PersonID,
				&order.BookID,
				&borrowDate,
				&returnDate,
				&actualReturnDate,
				&order.Status,
			)
			if err != nil {
				log.Printf("scan order: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan order"})
				return
			}

			order.BorrowDate = borrowDate.Format("2006-01-02")
			if returnDate.Valid {
				formattedReturnDate := returnDate.Time.Format("2006-01-02")
				order.ReturnDate = &formattedReturnDate
			}
			if actualReturnDate.Valid {
				formattedActualReturnDate := actualReturnDate.Time.Format("2006-01-02")
				order.ActualReturnDate = &formattedActualReturnDate
			}

			orders = append(orders, order)
		}

		if err := rows.Err(); err != nil {
			log.Printf("get all orders: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get orders"})
			return
		}

		c.JSON(http.StatusOK, orders)
	}
}

// GetOrderBookByID retrieves a specific order by ID
func GetOrderBookByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.Database()
		if db == nil {
			log.Println("database connection is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}

		orderID, err := strconv.Atoi(c.Param("id"))
		if err != nil || orderID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order_id"})
			return
		}

		query := `
            SELECT OrderID, PersonID, BookID, BorrowDate, ReturnDate, ActualReturnDate, Status
            FROM OrderBook
            WHERE OrderID = ?`
		var order OrderBook
		var borrowDate time.Time
		var returnDate, actualReturnDate sql.NullTime

		err = db.QueryRow(query, orderID).Scan(
			&order.OrderID,
			&order.PersonID,
			&order.BookID,
			&borrowDate,
			&returnDate,
			&actualReturnDate,
			&order.Status,
		)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		if err != nil {
			log.Printf("query order %d: %v", orderID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve order"})
			return
		}

		order.BorrowDate = borrowDate.Format("2006-01-02")
		if returnDate.Valid {
			dateStr := returnDate.Time.Format("2006-01-02")
			order.ReturnDate = &dateStr
		}
		if actualReturnDate.Valid {
			dateStr := actualReturnDate.Time.Format("2006-01-02")
			order.ActualReturnDate = &dateStr
		}

		c.JSON(http.StatusOK, order)
	}
}

// isValidStatus checks if the status is valid
func isValidStatus(status string) bool {
	validStatuses := map[string]bool{
		"Borrowed": true,
		"Returned": true,
		"Overdue":  true,
	}
	return validStatuses[status]
}
