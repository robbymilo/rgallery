package users

import (
	"log"

	"github.com/robbymilo/rgallery/pkg/database"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// ResetUsers removes all users and creates an account with username 'admin' and password 'admin'.
func ResetUsers(c Conf) error {
	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadWrite)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.Logger.Error("conn.Close error", "err", err)
		}
	}()

	const deleteStmt = "DELETE FROM users;"
	err = sqlitex.Execute(conn, deleteStmt, nil)
	if err != nil {
		log.Fatalf("Failed to delete rows: %v", err)
	}

	err = InitUser(c)
	if err != nil {
		return err
	}

	return nil
}
