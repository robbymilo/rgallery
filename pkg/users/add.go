package users

import (
	"errors"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/types"
	"golang.org/x/crypto/bcrypt"
	"zombiezen.com/go/sqlite"
)

type UserCredentials = types.UserCredentials
type Conf = types.Conf
type User = types.User

func AddUser(creds UserCredentials, c Conf) error {
	if creds.Username == "admin" {
		return errors.New("admin user may not be created")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing password: %w", err)
	}

	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadWrite)
	if err != nil {
		return fmt.Errorf("error opening sqlite db: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.Logger.Error("conn.Close error", "err", err)
		}
	}()

	// check for existing username
	stmt, err := conn.Prepare("SELECT 1 FROM users WHERE username = ?")
	if err != nil {
		return fmt.Errorf("error preparing select statement: %w", err)
	}

	stmt.BindText(1, creds.Username)

	if hasRow, err := stmt.Step(); err != nil {
		return fmt.Errorf("error checking for existing user: %w", err)
	} else if hasRow {
		return errors.New("user exists")
	}

	if finalizeErr := stmt.Finalize(); finalizeErr != nil {
		return fmt.Errorf("error finalizing select statement: %v", finalizeErr)
	}

	// insert new user
	insertStmt, err := conn.Prepare("INSERT INTO users (username, password, role) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("error preparing insert statement: %w", err)
	}

	insertStmt.BindText(1, creds.Username)
	insertStmt.BindText(2, string(hashedPassword))
	insertStmt.BindText(3, creds.Role)

	if _, err := insertStmt.Step(); err != nil {
		return fmt.Errorf("error inserting scan record: %v", err)
	}

	if finalizeErr := insertStmt.Finalize(); finalizeErr != nil {
		return fmt.Errorf("error finalizing insert statement: %v", finalizeErr)
	}

	// delete default admin user if necessary
	adminCheckStmt, err := conn.Prepare("SELECT username FROM users WHERE username = ?")
	if err != nil {
		return fmt.Errorf("error preparing select for admin user: %w", err)
	}

	adminCheckStmt.BindText(1, "admin")

	hasAdmin, err := adminCheckStmt.Step()
	if err != nil {
		return fmt.Errorf("error executing select for admin user: %w", err)
	}

	if hasAdmin {
		adminCreds := &UserCredentials{Username: "admin"}
		if err := RemoveUser(adminCreds, conn); err != nil {
			return fmt.Errorf("error removing admin user: %w", err)
		}
	}

	if finalizeErr := adminCheckStmt.Finalize(); finalizeErr != nil {
		return fmt.Errorf("error finalizing admin check statement: %v", finalizeErr)
	}

	return nil
}
