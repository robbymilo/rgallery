package queries

import (
	"context"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/hash"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// GetTag returns media items with a given exif tag.
func GetTag(offset int, direction string, pageSize int, group, name string, c Conf) ([]Media, error) {
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

	query := fmt.Sprintf(`SELECT DISTINCT hash, i.path, i.subject, i.width, i.height, i.ratio, i.padding, i.date, i.modified, i.folder, i.rating, i.shutterspeed, i.aperture, i.iso, i.lens, i.camera, i.focallength, i.altitude, i.latitude, i.longitude, i.mediatype, i.focusdistance, i.focallength35, i.color, i.location, i.description, i.title, i.software, i.offset, i.rotation FROM media i JOIN images_tags i_a ON i.hash = i_a.image_id WHERE i_a.tag_id =? GROUP BY i.date ORDER BY i.date %s LIMIT %d OFFSET %d`, direction, pageSize, offset)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing SELECT statement: %v", err)
	}

	stmt.BindInt64(1, int64(hash.GetHash(name)))

	result, err := parseMediaRows(stmt, c)
	if err != nil {
		return nil, err
	}

	err = stmt.Finalize()
	if err != nil {
		return nil, fmt.Errorf("error finalizing statement: %v", err)
	}

	return result, err

}

// GetTotalOfTag returns the number of media items with a given exif tag.
func GetTotalOfTag(group string, c Conf) (int, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return 0, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("error closing pool", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return 0, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	query := `SELECT COUNT(distinct date) FROM media i JOIN images_tags i_a ON i.hash = i_a.image_id WHERE i_a.tag_id =?`
	stmt, err := conn.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("error preparing query: %v", err)
	}

	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

	stmt.BindInt64(1, int64(hash.GetHash(group)))

	var total int
	hasRow, err := stmt.Step()
	if err != nil {
		return 0, fmt.Errorf("error executing query: %v", err)
	}
	if hasRow {
		total = int(stmt.ColumnInt64(0))
	}

	return total, nil
}

// GetTagTitle returns the title of a tag given the tag's key.
func GetTagTitle(key string, c Conf) (string, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return "", fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("error closing pool", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	query := `SELECT value FROM tags WHERE key =?`
	stmt, err := conn.Prepare(query)
	if err != nil {
		return "", fmt.Errorf("error preparing query: %v", err)
	}

	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

	stmt.BindText(1, key)

	var value string
	hasRow, err := stmt.Step()
	if err != nil {
		return "", fmt.Errorf("error executing query: %v", err)
	}
	if hasRow {
		value = stmt.ColumnText(0)
	}

	return value, nil
}
