package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/robbymilo/rgallery/pkg/database"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type ScanError struct {
	Path     string    `json:"path"`
	Modified time.Time `json:"modified"`
	Error    string    `json:"error"`
}

type ScanErrorsPage struct {
	Items []ScanError `json:"items"`
	Total int         `json:"total"`
}

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

// ClearScanError removes the retry marker after the file has been ingested
// successfully. A retry must not continue to be suppressed by an old error.
func ClearScanError(path string, c Conf) error {
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

	stmt, err := conn.Prepare("DELETE FROM scan_errors WHERE path = ?")
	if err != nil {
		return fmt.Errorf("error preparing scan error delete: %v", err)
	}
	defer func() {
		if err := stmt.Finalize(); err != nil {
			c.Logger.Error("error finalizing statement", "error", err)
		}
	}()
	stmt.BindText(1, path)
	if _, err := stmt.Step(); err != nil {
		return fmt.Errorf("error deleting scan error record: %v", err)
	}
	return nil
}

// ResetScanError removes one failed path so the next scan will try it again.
func ResetScanError(path string, c Conf) error {
	if path == "" {
		return fmt.Errorf("scan error path cannot be empty")
	}
	return ClearScanError(path, c)
}

// ResetAllScanErrors removes all failed paths so the next scan will retry them.
func ResetAllScanErrors(c Conf) error {
	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadWrite)
	if err != nil {
		return fmt.Errorf("error opening sqlite db: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			c.Logger.Error("error closing sqlite connection", "err", closeErr)
		}
	}()
	return sqlitex.Execute(conn, "DELETE FROM scan_errors", nil)
}

// GetScanErrorsPage returns scan errors in stable path order.
func GetScanErrorsPage(c Conf, page, pageSize int) (ScanErrorsPage, error) {
	if page < 1 || pageSize < 1 {
		return ScanErrorsPage{}, fmt.Errorf("page and page size must be positive")
	}
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags: sqlite.OpenReadOnly, PoolSize: 1,
	})
	if err != nil {
		return ScanErrorsPage{}, fmt.Errorf("error opening sqlite db pool: %w", err)
	}
	defer func() {
		if closeErr := pool.Close(); closeErr != nil {
			c.Logger.Error("error closing pool", "err", closeErr)
		}
	}()
	conn, err := pool.Take(context.Background())
	if err != nil {
		return ScanErrorsPage{}, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	result := ScanErrorsPage{Items: []ScanError{}}
	if err := sqlitex.Execute(conn, "SELECT COUNT(*) FROM scan_errors", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			result.Total = stmt.ColumnInt(0)
			return nil
		},
	}); err != nil {
		return ScanErrorsPage{}, err
	}
	if err := sqlitex.Execute(conn, "SELECT path, modified, error FROM scan_errors ORDER BY path ASC LIMIT ? OFFSET ?", &sqlitex.ExecOptions{
		Args: []interface{}{pageSize, (page - 1) * pageSize},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			modified, parseErr := time.Parse(time.RFC3339, stmt.ColumnText(1))
			if parseErr != nil {
				return fmt.Errorf("error parsing scan error time for %q: %w", stmt.ColumnText(0), parseErr)
			}
			result.Items = append(result.Items, ScanError{Path: stmt.ColumnText(0), Modified: modified, Error: stmt.ColumnText(2)})
			return nil
		},
	}); err != nil {
		return ScanErrorsPage{}, err
	}
	return result, nil
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
