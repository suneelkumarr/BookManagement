package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/microsoft/go-mssqldb"
)

var db *sql.DB // Global variable to store the database connection

// Database initializes and returns a database connection
func Database() *sql.DB {
	if db == nil { // Check if the connection already exists
		// Connection parameters
		user := "sa"
		password := "goteg@123"
		server := `.\SQLEXPRESS` // Use raw string literal for backslash
		databaseName := "BookManagement"
		port := "1433"

		// Connection string format for go-mssqldb
		connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s;encrypt=true;TrustServerCertificate=true",
			server, user, password, port, databaseName)

		fmt.Println("Attempting to connect with:", connString)

		// Open the connection
		var err error
		db, err = sql.Open("mssql", connString)
		if err != nil {
			log.Fatalf("Error creating connection pool: %v", err)
		}

		// Test the connection
		err = db.Ping()
		if err != nil {
			log.Fatalf("Ping failed: %v", err)
		}

		fmt.Println("Connected to MSSQL Server successfully!")
	}
	return db
}
