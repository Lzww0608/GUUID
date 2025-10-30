package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/lab2439/guuid"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// User represents a user in the database
type User struct {
	ID        guuid.UUID
	Username  string
	Email     string
	CreatedAt int64
}

func main() {
	fmt.Println("=== GUUID Database Integration Example ===\n")

	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table
	createTable(db)

	// Insert users
	fmt.Println("1. Inserting users with UUIDv7 primary keys:")
	users := []User{
		{ID: guuid.Must(guuid.New()), Username: "alice", Email: "alice@example.com"},
		{ID: guuid.Must(guuid.New()), Username: "bob", Email: "bob@example.com"},
		{ID: guuid.Must(guuid.New()), Username: "charlie", Email: "charlie@example.com"},
	}

	for _, user := range users {
		user.CreatedAt = user.ID.Timestamp()
		insertUser(db, user)
		fmt.Printf("   Inserted: %s (%s) - ID: %s\n", user.Username, user.Email, user.ID)
	}
	fmt.Println()

	// Query users
	fmt.Println("2. Querying all users (ordered by UUID/time):")
	queriedUsers := queryAllUsers(db)
	for i, user := range queriedUsers {
		fmt.Printf("   %d. %s - %s (Created: %d ms)\n", i+1, user.ID, user.Username, user.CreatedAt)
	}
	fmt.Println()

	// Query single user
	fmt.Println("3. Querying user by ID:")
	if len(users) > 0 {
		user := queryUserByID(db, users[0].ID)
		if user != nil {
			fmt.Printf("   Found: %s (%s)\n", user.Username, user.Email)
		}
	}
	fmt.Println()

	// Update user
	fmt.Println("4. Updating user:")
	if len(users) > 0 {
		users[0].Email = "alice.updated@example.com"
		updateUser(db, users[0])
		updated := queryUserByID(db, users[0].ID)
		if updated != nil {
			fmt.Printf("   Updated email: %s\n", updated.Email)
		}
	}
	fmt.Println()

	// Delete user
	fmt.Println("5. Deleting user:")
	if len(users) > 1 {
		deleteUser(db, users[1].ID)
		fmt.Printf("   Deleted user: %s\n", users[1].Username)
		remaining := queryAllUsers(db)
		fmt.Printf("   Remaining users: %d\n", len(remaining))
	}
}

func createTable(db *sql.DB) {
	query := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func insertUser(db *sql.DB, user User) {
	query := `INSERT INTO users (id, username, email, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, user.ID, user.Username, user.Email, user.CreatedAt)
	if err != nil {
		log.Fatal(err)
	}
}

func queryAllUsers(db *sql.DB) []User {
	query := `SELECT id, username, email, created_at FROM users ORDER BY id`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
		if err != nil {
			log.Fatal(err)
		}
		users = append(users, user)
	}
	return users
}

func queryUserByID(db *sql.DB, id guuid.UUID) *User {
	query := `SELECT id, username, email, created_at FROM users WHERE id = ?`
	row := db.QueryRow(query, id)

	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Fatal(err)
	}
	return &user
}

func updateUser(db *sql.DB, user User) {
	query := `UPDATE users SET email = ? WHERE id = ?`
	_, err := db.Exec(query, user.Email, user.ID)
	if err != nil {
		log.Fatal(err)
	}
}

func deleteUser(db *sql.DB, id guuid.UUID) {
	query := `DELETE FROM users WHERE id = ?`
	_, err := db.Exec(query, id)
	if err != nil {
		log.Fatal(err)
	}
}
