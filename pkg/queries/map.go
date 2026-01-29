package queries

import (
	"context"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/types"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type MapItem = types.MapItem

// GetMapItems returns all media items' coordinates.
func GetMapItems(c Conf) ([]MapItem, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("error closing pool: %v", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	query := `
			SELECT DISTINCT hash, latitude, longitude, date
			FROM media
			WHERE latitude != 0.0
				AND longitude != 0.0
				AND date != '0001-01-01T00:00:00.000Z'
			GROUP BY date`

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing query: %v", err)
	}
	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

	var mapItems []MapItem
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("error stepping through query results: %v", err)
		}
		if !hasRow {
			break
		}

		var item MapItem
		item = append(item, stmt.ColumnFloat(1))
		item = append(item, stmt.ColumnFloat(2))
		item = append(item, stmt.ColumnFloat(0))

		mapItems = append(mapItems, item)
	}

	return mapItems, nil
}
