package database

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/robbymilo/rgallery/pkg/types"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema []byte

type Conf = types.Conf

func CreateDB(c Conf) {
	// Create db filepath
	path, err := filepath.Abs(c.Data)
	if err != nil {
		c.Logger.Error("error creating db filepath", "error", err)
		return
	}

	// Check if the data directory exists and create if it doesn't
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		c.Logger.Info("data directory does not exist, creating at: " + path)
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			c.Logger.Error("error creating data directory", "error", err)
			return
		}
	}

	c.Logger.Info("database located at " + NewConnectionString(c))

	db, err := sql.Open("sqlite", NewConnectionString(c))
	if err != nil {
		c.Logger.Error("error opening sqlite db", "error", err)
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			c.Logger.Error("db.Close error", "err", err)
		}
	}()

	c.Logger.Info("applying sqlite schema")

	if err := applySchema(db); err != nil {
		c.Logger.Error("error applying schema", "error", err)
		return
	}

	c.Logger.Info("sqlite schema applied")

	if err := renamePathColumn(db); err != nil {
		c.Logger.Error("error renaming path column", "error", err)
		return
	}

	var journal_mode string
	err = db.QueryRow("PRAGMA journal_mode;").Scan(&journal_mode)
	if err != nil {
		log.Fatal(err)
	}

	c.Logger.Info("Current journal_mode: " + journal_mode)

	setSchemaVersion(db, 20241120)
}

// applySchema applies the schema to the database
func applySchema(db *sql.DB) error {
	statement, err := db.Prepare(string(schema))
	if err != nil {
		return fmt.Errorf("error preparing schema statement: %w", err)
	}
	defer func() {
		if err := statement.Close(); err != nil {
			// No context available here, so just log to standard error
			fmt.Println("statement.Close error:", err)
		}
	}()

	_, err = statement.Exec()
	if err != nil {
		return fmt.Errorf("error executing schema statement: %w", err)
	}

	return nil
}

// setSchemaVersion updates or inserts the schema version in the schema table
func setSchemaVersion(db *sql.DB, version int) {
	statement, err := db.Prepare(`INSERT OR REPLACE INTO schema(key, value) VALUES (?, ?);`)
	if err != nil {
		log.Fatal("error preparing schema version statement: ", err.Error())
	}
	defer func() {
		if err := statement.Close(); err != nil {
			fmt.Println("statement.Close error:", err)
		}
	}()

	_, err = statement.Exec("version", version)
	if err != nil {
		log.Fatal("error executing schema version statement: ", err.Error())
	}
}

// renamePathColumn renames the 'name' column to 'path' in the 'media' table
func renamePathColumn(db *sql.DB) error {
	var columnExists int
	query := `SELECT COUNT(*) FROM pragma_table_info('media') WHERE name = 'name';`
	err := db.QueryRow(query).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check for column existence: %w", err)
	}

	if columnExists == 0 {
		return nil
	}

	_, err = db.Exec(`ALTER TABLE media RENAME COLUMN name TO path;`)
	if err != nil {
		return fmt.Errorf("failed to rename column: %w", err)
	}

	fmt.Println("Column 'name' successfully renamed to 'path'.")
	return nil
}

func NewConnectionString(c Conf) string {
	path, err := filepath.Abs(c.Data)
	if err != nil {
		c.Logger.Error("error parsing data dir", "error", err)
	}

	return filepath.Join(path, "/sqlite.db?journal_mode=WAL&synchronous=1&page_size=32768&socket_timeout=50000&cache_size=1000000000&foreign_keys=true&busy_timeout=5000")
}

func NewSqlConnectionString(c Conf) string {
	path, err := filepath.Abs(c.Data)
	if err != nil {
		c.Logger.Error("error parsing data dir", "error", err)
	}

	return filepath.Join(path, "/sqlite.db")
}

func Columns() string {
	return `hash, path, subject, width, height, ratio, padding, date, modified, folder, rating, shutterspeed, aperture, iso, lens, camera, focallength, altitude, latitude, longitude, mediatype, focusdistance, focallength35, color, location, description, title, software, offset, rotation`
}
