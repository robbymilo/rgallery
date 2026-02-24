package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/robbymilo/rgallery/pkg/database"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// TrackScanError records a file scanning error in the database
// This allows for later reporting and retry mechanisms
func TrackScanError(path string, lastScanTime time.Time, error error, c Conf) error {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadWrite,
		PoolSize: 1,
	})
	if err != nil {
		return fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("error closing pool", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	// Insert the error into the scan_errors table
	stmt, prepErr := conn.Prepare("INSERT OR REPLACE INTO scan_errors (path, modified, error) VALUES (?, ?, ?)")
	if prepErr != nil {
		return fmt.Errorf("error preparing insert statement: %v", prepErr)
	}

	stmt.BindText(1, path)
	stmt.BindText(2, lastScanTime.Format(time.RFC3339))
	stmt.BindText(3, fmt.Sprint(error))

	if _, stepErr := stmt.Step(); stepErr != nil {
		return fmt.Errorf("error inserting scan error record: %v", stepErr)
	}

	finalErr := stmt.Finalize()
	if finalErr != nil {
		return fmt.Errorf("error finalizing statement: %v", finalErr)
	}

	return nil
}

// GetScanErrors retrieves all scan errors from the database
func GetScanErrors(c Conf) (map[string]time.Time, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})

	if err != nil {
		return nil, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("error closing pool", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	stmt, prepErr := conn.Prepare("SELECT path, modified FROM scan_errors")
	if prepErr != nil {
		return nil, prepErr
	}

	defer func() {
		if err := stmt.Finalize(); err != nil {
			c.Logger.Error("error finalizing statement", "error", err)
		}
	}()

	scanErrors := make(map[string]time.Time)

	for {
		hasRow, stepErr := stmt.Step()
		if stepErr != nil {
			return nil, stepErr
		}

		if !hasRow {
			break
		}

		path := stmt.ColumnText(0)
		lastScanTime := stmt.ColumnText(1)

		time, err := time.Parse(time.RFC3339, lastScanTime)
		if err != nil {
			return nil, fmt.Errorf("error parsing last scan time: %v", err)
		}

		scanErrors[path] = time

	}

	return scanErrors, nil
}
