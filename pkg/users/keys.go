package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/types"
	"golang.org/x/crypto/bcrypt"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type ApiCredentials = types.ApiCredentials

// AddKey adds an API key.
func AddKey(creds *ApiCredentials, c Conf) (string, error) {

	if creds.Name == "" {
		return "", errors.New("key must have a name")
	}

	// create API key value
	creds.Key = uuid.New().String()
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(creds.Key), 8)
	if err != nil {
		return "", err
	}

	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadWrite)
	if err != nil {
		return "", fmt.Errorf("error opening sqlite db: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.Logger.Error("conn.Close error", "err", err)
		}
	}()

	// check for existing key
	stmt, err := conn.Prepare("SELECT 1 FROM keys WHERE name=?")
	if err != nil {
		return "", fmt.Errorf("error preparing select statement: %w", err)
	}

	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing select statement: %v", finalizeErr)
		}
	}()

	stmt.BindText(1, creds.Name)

	var hasRow bool
	if hasRow, err := stmt.Step(); err != nil {
		return "", fmt.Errorf("error checking for existing key: %w", err)
	} else if hasRow {
		return "", errors.New("key exists")
	}

	if hasRow {
		return "", errors.New("error adding key, key exists")
	}

	// insert new key
	insertStmt, err := conn.Prepare("INSERT INTO keys VALUES (?, ?, ?)")
	if err != nil {
		return "", fmt.Errorf("error preparing insert statement: %w", err)
	}

	defer func() {
		if finalizeErr := insertStmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing insert statement: %v", finalizeErr)
		}
	}()

	insertStmt.BindText(1, creds.Name)
	insertStmt.BindText(2, string(hashedKey))
	insertStmt.BindText(3, time.Now().UTC().Format(time.RFC3339))

	if _, err := insertStmt.Step(); err != nil {
		return "", fmt.Errorf("error inserting scan record: %v", err)
	}

	return creds.Key, nil

}

// RemoveKey removes an API key.
func RemoveKey(creds *ApiCredentials, c Conf) error {

	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadWrite)
	if err != nil {
		return fmt.Errorf("error opening sqlite db: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.Logger.Error("conn.Close error", "err", err)
		}
	}()

	stmt, err := conn.Prepare("DELETE FROM keys WHERE name =?")
	if err != nil {
		return fmt.Errorf("error preparing delete statement: %w", err)
	}

	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing insert statement: %v", finalizeErr)
		}
	}()

	stmt.BindText(1, creds.Name)

	_, err = stmt.Step()
	if err != nil {
		return fmt.Errorf("error executing delete statement: %w", err)
	}

	return nil

}

// GetKeyNames returns a list of all API key names.
func GetKeyNames(c Conf) ([]ApiCredentials, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("pool.Close error", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	stmt, err := conn.Prepare("SELECT name, created FROM keys")
	if err != nil {
		return nil, fmt.Errorf("error preparing SELECT statement: %v", err)
	}

	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

	keys := make([]ApiCredentials, 0)
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("error stepping through result set: %v", err)
		}
		if !hasRow {
			break
		}

		t, err := time.Parse("2006-01-02T15:04:05Z", stmt.ColumnText(1))
		if err != nil {
			return []ApiCredentials{}, fmt.Errorf("error parsing date for key: %v", err)
		}

		keys = append(keys, ApiCredentials{
			Name:    stmt.ColumnText(0),
			Created: t,
		})
	}

	return keys, nil
}
