package controllers

import (
	"database/sql"
	"fmt"
	database "go-crud-api/config"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Book struct {
	BookID         int     `json:"bookid"`
	TypeOfBook     string  `json:"typeofbook"`
	BookName       string  `json:"bookname"`
	BookAuthorName string  `json:"bookauthorname"`
	IsAvailable    bool    `json:"isavailable"`
	BookQuantity   int     `json:"bookquantity"`
	BookPrice      float64 `json:"bookprice"`
}

// UpdateBookInput represents the input for updating a book
type UpdateBookInput struct {
	TypeOfBook     *string  `json:"typeofbook"`
	BookName       *string  `json:"bookname"`
	BookAuthorName *string  `json:"bookauthorname"`
	IsAvailable    *bool    `json:"isavailable"`
	BookQuantity   *int     `json:"bookquantity"`
	BookPrice      *float64 `json:"bookprice"`
}

func CreateBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the database connection
		db := database.Database()

		// Parse the request body into a Book struct
		var newBook Book
		if err := c.ShouldBindJSON(&newBook); err != nil {
			log.Printf("Invalid request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Validate required fields
		if newBook.BookName == "" || newBook.TypeOfBook == "" || newBook.BookAuthorName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Book name, type of book, and author name are required fields"})
			return
		}

		// Check if a book with the same name and author already exists
		var existingID int
		checkQuery := "SELECT BookID FROM Book WHERE bookName = ? AND bookAuthorName = ?"
		err := db.QueryRow(checkQuery, newBook.BookName, newBook.BookAuthorName).Scan(&existingID)

		if err == nil {
			// Book already exists, return conflict status
			c.JSON(http.StatusConflict, gin.H{"error": "A book with this name and author already exists", "existingID": existingID})
			return
		} else if err != sql.ErrNoRows {
			// Unexpected database error
			log.Printf("Error checking for existing book: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to verify book uniqueness. Please try again later."})
			return
		}

		// If we reach here, the book does not exist, so we can proceed with insertion
		// Prepare the SQL query to insert a new book and get the inserted ID using MSSQL syntax
		query := `
			INSERT INTO Book (typeOfBook, bookName, bookAuthorName, isAvailable, bookQuantity, bookPrice) 
			VALUES (?, ?, ?, ?, ?, ?);
			SELECT SCOPE_IDENTITY() AS ID;
		`

		// Execute the query with parameterized values using standard placeholders
		var id int
		err = db.QueryRow(query,
			newBook.TypeOfBook,
			newBook.BookName,
			newBook.BookAuthorName,
			newBook.IsAvailable,
			newBook.BookQuantity,
			newBook.BookPrice).Scan(&id)

		if err != nil {
			log.Printf("Failed to execute insert query: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create book. Please try again later."})
			return
		}

		// Fetch the newly created book data
		var bookDetails Book
		err = db.QueryRow("SELECT BookID, typeOfBook, bookName, bookAuthorName, isAvailable, bookQuantity, bookPrice FROM Book WHERE BookID = ?", id).Scan(
			&bookDetails.BookID,
			&bookDetails.TypeOfBook,
			&bookDetails.BookName,
			&bookDetails.BookAuthorName,
			&bookDetails.IsAvailable,
			&bookDetails.BookQuantity,
			&bookDetails.BookPrice,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("No record found for BookID %d", id)
				c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
			} else {
				log.Printf("Failed to fetch created book data: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to fetch created book data. Please try again later."})
			}
			return
		}

		// Send the response with the created book data
		c.JSON(http.StatusCreated, gin.H{"data": bookDetails})
	}
}

func GetBooks() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the database connection
		db := database.Database()

		// Prepare the SQL query to fetch all books
		query := "SELECT BookID, typeOfBook, bookName, bookAuthorName, isAvailable, bookQuantity, bookPrice FROM Book"

		// Execute the query and scan the results into a slice of Book structs
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Failed to execute query: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to fetch books. Please try again later."})
			return
		}
		defer rows.Close() // Ensure rows are closed after processing

		// Process the query results
		var books []Book
		for rows.Next() {
			var book Book
			err := rows.Scan(&book.BookID, &book.TypeOfBook, &book.BookName, &book.BookAuthorName, &book.IsAvailable, &book.BookQuantity, &book.BookPrice)
			if err != nil {
				log.Printf("Failed to scan row: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to process book data. Please try again later."})
				return
			}
			books = append(books, book)
		}

		// Check for errors during iteration
		if err := rows.Err(); err != nil {
			log.Printf("Error during row iteration: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to process book data. Please try again later."})
			return
		}

		// Send the response with the fetched books
		c.JSON(http.StatusOK, gin.H{"data": books})
	}
}

func GetBookByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the database connection
		db := database.Database()

		// Get the book ID from the URL parameters
		bookID := c.Param("id")

		// Prepare the SQL query to fetch a single book by ID
		query := "SELECT BookID, typeOfBook, bookName, bookAuthorName, isAvailable, bookQuantity, bookPrice FROM Book WHERE BookID = ?"

		// Execute the query with a parameterized statement
		var book Book
		err := db.QueryRow(query, bookID).Scan(&book.BookID, &book.TypeOfBook, &book.BookName, &book.BookAuthorName, &book.IsAvailable, &book.BookQuantity, &book.BookPrice)
		if err != nil {
			if err == sql.ErrNoRows {
				// No rows found for the given ID
				c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
			} else {
				// Handle other errors
				log.Printf("Failed to execute query: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to fetch book data. Please try again later."})
			}
			return
		}

		// Send the response with the book data
		c.JSON(http.StatusOK, gin.H{"data": book})
	}
}

func GetBookByName() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the database connection (assumes database.Database() returns a *sql.DB)
		db := database.Database()

		// Get the book name from the URL parameters
		bookName := strings.TrimSpace(c.Param("name"))
		if bookName == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Book name cannot be empty",
				"data":  nil,
			})
			return
		}

		// Add wildcards for partial matching and handle case-insensitive search
		searchName := "%" + bookName + "%"
		query := `
            SELECT BookID, typeOfBook, bookName, bookAuthorName, isAvailable, bookQuantity, bookPrice
            FROM Book
            WHERE LOWER(bookName) LIKE LOWER(?)`

		// Execute the query (using ? for MSSQL compatibility)
		rows, err := db.Query(query, searchName)
		if err != nil {
			log.Printf("Failed to execute query: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Unable to fetch book data",
				"data":  nil,
			})
			return
		}
		defer rows.Close()

		// Collect books
		var books []Book
		for rows.Next() {
			var book Book
			err := rows.Scan(&book.BookID, &book.TypeOfBook, &book.BookName, &book.BookAuthorName, &book.IsAvailable, &book.BookQuantity, &book.BookPrice)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Error processing book data",
					"data":  nil,
				})
				return
			}
			books = append(books, book)
		}

		// Check for iteration errors
		if err = rows.Err(); err != nil {
			log.Printf("Error iterating rows: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error processing book data",
				"data":  nil,
			})
			return
		}

		// Handle no results
		if len(books) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No books found with the given name",
				"data":  nil,
			})
			return
		}

		// Return the books
		c.JSON(http.StatusOK, gin.H{
			"error": nil,
			"data":  books,
		})
	}
}

func GetBookByAuthor() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the database connection (assumes database.Database() returns a *sql.DB)
		db := database.Database()

		// Get the book author from the URL parameters
		bookAuthorName := strings.TrimSpace(c.Param("author"))
		if bookAuthorName == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Book author name cannot be empty",
				"data":  nil,
			})
			return
		}

		// Add wildcards for partial matching and handle case-insensitive search
		searchAuthor := "%" + bookAuthorName + "%"
		query := `SELECT BookID, typeOfBook, bookName, bookAuthorName, isAvailable, bookQuantity, bookPrice
            FROM Book
            WHERE LOWER(bookAuthorName) LIKE LOWER(?)`

		// Execute the query (using ? for MSSQL compatibility)
		rows, err := db.Query(query, searchAuthor)
		if err != nil {
			log.Printf("Failed to execute query: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Unable to fetch book data",
				"data":  nil,
			})
			return
		}
		defer rows.Close()

		// Collect books
		var books []Book
		for rows.Next() {
			var book Book
			err := rows.Scan(&book.BookID, &book.TypeOfBook, &book.BookName, &book.BookAuthorName, &book.IsAvailable, &book.BookQuantity, &book.BookPrice)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Error processing book data",
					"data":  nil,
				})
				return
			}
			books = append(books, book)
		}

		// Check for iteration errors
		if err = rows.Err(); err != nil {
			log.Printf("Error iterating rows: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error processing book data",
				"data":  nil,
			})
			return
		}

		// Handle no results
		if len(books) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No books found with the given author",
				"data":  nil,
			})
			return
		}

		// Return the books
		c.JSON(http.StatusOK, gin.H{
			"error": nil,
			"data":  books,
		})
	}
}

func GetBookByType() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the database connection (assumes database.Database() returns a *sql.DB)
		db := database.Database()

		// Get the book type from the URL parameters
		typeOfBook := strings.TrimSpace(c.Param("type"))
		if typeOfBook == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Book type cannot be empty",
				"data":  nil,
			})
			return
		}

		// Add wildcards for partial matching and handle case-insensitive search
		searchType := "%" + typeOfBook + "%"
		query := `SELECT BookID, typeOfBook, bookName, bookAuthorName, isAvailable, bookQuantity, bookPrice
					     FROM Book
            WHERE LOWER(typeOfBook) LIKE LOWER(?)`

		// Execute the query (using ? for MSSQL compatibility)
		rows, err := db.Query(query, searchType)
		if err != nil {
			log.Printf("Failed to execute query: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Unable to fetch book data",
				"data":  nil,
			})
			return
		}
		defer rows.Close()

		// Collect books
		var books []Book
		for rows.Next() {
			var book Book
			err := rows.Scan(&book.BookID, &book.TypeOfBook, &book.BookName, &book.BookAuthorName, &book.IsAvailable, &book.BookQuantity, &book.BookPrice)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Error processing book data",
					"data":  nil,
				})
				return
			}
			books = append(books, book)
		}

		// Check for iteration errors
		if err = rows.Err(); err != nil {
			log.Printf("Error iterating rows: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error processing book data",
				"data":  nil,
			})
			return
		}

		// Handle no results
		if len(books) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No books found with the given type",
				"data":  nil,
			})
			return
		}

		// Return the books
		c.JSON(http.StatusOK, gin.H{
			"error": nil,
			"data":  books,
		})
	}
}

// GetBookByAvailability handles fetching books by availability status
func GetBookByAvailability() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log the full request URL for debugging
		log.Printf("Handling request for URL: %s", c.Request.URL.Path)

		// Get the database connection (assumes database.Database() returns a *sql.DB)
		db := database.Database()

		// Get the availability status from the URL parameters
		availability := strings.TrimSpace(c.Param("isAvailable"))
		log.Printf("Received availability parameter: %q", availability)

		if availability == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Availability status cannot be empty",
				"data":  nil,
			})
			return
		}

		// Convert availability string to boolean (0 or 1 for MSSQL BIT)
		var isAvailable int
		availability = strings.ToLower(availability)
		switch availability {
		case "true", "1":
			isAvailable = 1
		case "false", "0":
			isAvailable = 0
		default:
			log.Printf("Invalid availability value: %s", availability)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid availability status. Use 'true', 'false', '1', or '0'",
				"data":  nil,
			})
			return
		}

		// Query for books with the specified availability
		query := `
            SELECT BookID, typeOfBook, bookName, bookAuthorName, isAvailable, bookQuantity, bookPrice
            FROM Book
            WHERE isAvailable = ?`

		// Execute the query
		rows, err := db.Query(query, isAvailable)
		if err != nil {
			log.Printf("Failed to execute query: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Unable to fetch book data",
				"data":  nil,
			})
			return
		}
		defer rows.Close()

		// Collect books
		var books []Book
		for rows.Next() {
			var book Book
			err := rows.Scan(&book.BookID, &book.TypeOfBook, &book.BookName, &book.BookAuthorName, &book.IsAvailable, &book.BookQuantity, &book.BookPrice)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Error processing book data",
					"data":  nil,
				})
				return
			}
			books = append(books, book)
		}

		// Check for iteration errors
		if err = rows.Err(); err != nil {
			log.Printf("Error iterating rows: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error processing book data",
				"data":  nil,
			})
			return
		}

		// Handle no results
		if len(books) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No books found with the given availability",
				"data":  nil,
			})
			return
		}

		// Return the books
		c.JSON(http.StatusOK, gin.H{
			"error": nil,
			"data":  books,
		})
	}
}

// UpdateBook dynamically updates a book's fields based on provided input
func UpdateBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the database connection
		db := database.Database()

		// Get the book ID from URL parameters
		bookID := strings.TrimSpace(c.Param("id"))
		if bookID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Book ID cannot be empty",
				"data":  nil,
			})
			return
		}

		// Parse the request body into UpdateBookInput
		var input UpdateBookInput
		if err := c.ShouldBindJSON(&input); err != nil {
			log.Printf("Failed to parse request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
				"data":  nil,
			})
			return
		}
		// Build the dynamic SQL query
		var setClauses []string
		var args []interface{}

		if input.TypeOfBook != nil {
			setClauses = append(setClauses, "typeOfBook = ?")
			args = append(args, *input.TypeOfBook)
		}
		if input.BookName != nil {
			setClauses = append(setClauses, "bookName = ?")
			args = append(args, *input.BookName)
		}
		if input.BookAuthorName != nil {
			setClauses = append(setClauses, "bookAuthorName = ?")
			args = append(args, *input.BookAuthorName)
		}
		if input.IsAvailable != nil {
			setClauses = append(setClauses, "isAvailable = ?")
			args = append(args, *input.IsAvailable)
		}
		if input.BookQuantity != nil {
			setClauses = append(setClauses, "bookQuantity = ?")
			args = append(args, *input.BookQuantity)
		}
		if input.BookPrice != nil {
			setClauses = append(setClauses, "bookPrice = ?")
			args = append(args, *input.BookPrice)
		}

		// Check if any fields were provided for update
		if len(setClauses) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No fields provided for update",
				"data":  nil,
			})
			return
		}

		// Construct the SQL query
		query := fmt.Sprintf(
			"UPDATE Book SET %s WHERE BookID = ?",
			strings.Join(setClauses, ", "),
		)
		args = append(args, bookID)

		// Log the query for debugging (avoid logging args to prevent sensitive data exposure)
		log.Printf("Executing query: %s", query)

		// Execute the update
		result, err := db.Exec(query, args...)
		if err != nil {
			log.Printf("Failed to execute update query: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update book",
				"data":  nil,
			})
			return
		}

		// Check if any rows were affected
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("Failed to check rows affected: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify update",
				"data":  nil,
			})
			return
		}

		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Book not found",
				"data":  nil,
			})
			return
		}

		// Return success response
		c.JSON(http.StatusOK, gin.H{
			"error": nil,
			"data":  gin.H{"message": "Book updated successfully"},
		})
	}
}
