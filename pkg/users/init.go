package users

import (
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"golang.org/x/crypto/bcrypt"
	"zombiezen.com/go/sqlite"
)

// InitUser adds a user with username "admin" and password "admin" if no users exist.
func InitUser(c Conf) error {
	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadWrite)
	if err != nil {
		return fmt.Errorf("error opening sqlite db: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.Logger.Error("conn.Close error", "err", err)
		}
	}()

	// check if admin user already exists
	stmt, err := conn.Prepare("SELECT 1 FROM users LIMIT 1")
	if err != nil {
		return fmt.Errorf("error preparing select statement: %w", err)
	}

	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

	if hasRow, err := stmt.Step(); err != nil {
		return fmt.Errorf("error checking existing users: %w", err)
	} else if hasRow {
		return nil
	}

	// generate hashed password of "admin" for the "admin" user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error creating admin password: %w", err)
	}

	// insert "admin" user into the database
	insertStmt, err := conn.Prepare("INSERT INTO users (username, password, role) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("error preparing insert statement: %w", err)
	}

	defer func() {
		if finalizeErr := insertStmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()
	insertStmt.BindText(1, "admin")
	insertStmt.BindText(2, string(hashedPassword))
	insertStmt.BindText(3, "admin")

	if _, err := insertStmt.Step(); err != nil {
		return fmt.Errorf("error inserting admin user: %w", err)
	}

	return nil
}
