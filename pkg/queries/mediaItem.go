package queries

import (
	"context"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/sizes"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// GetSingleMediaItem retrieves a single media item by hash from the database.
func GetSingleMediaItem(hash uint32, c Conf) (Media, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return Media{}, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("error closing pool", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return Media{}, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	query := fmt.Sprintf(`SELECT %s FROM media WHERE hash = ?;`, columns)
	stmt, err := conn.Prepare(query)
	if err != nil {
		return Media{}, fmt.Errorf("error preparing query for media item: %v", err)
	}

	stmt.BindInt64(1, int64(hash))

	hasRow, err := stmt.Step()
	if err != nil {
		return Media{}, fmt.Errorf("error stepping through result set: %v", err)
	}
	if !hasRow {
		return Media{}, nil
	}

	mediaData := DatabaseMedia{}

	mediaData.Hash = uint32(stmt.ColumnInt64(0))
	mediaData.Path = stmt.ColumnText(1)
	mediaData.Subject = stmt.ColumnText(2)
	mediaData.Width = int(stmt.ColumnInt64(3))
	mediaData.Height = int(stmt.ColumnInt64(4))
	mediaData.Ratio = float32(stmt.ColumnFloat(5))
	mediaData.Padding = float32(stmt.ColumnFloat(6))
	mediaData.Date = stmt.ColumnText(7)
	// mediaData.Day = stmt.ColumnText(8)
	mediaData.Modified = stmt.ColumnText(8)
	mediaData.Folder = stmt.ColumnText(9)
	mediaData.Rating = stmt.ColumnFloat(10)
	mediaData.ShutterSpeed = stmt.ColumnText(11)
	mediaData.Aperture = stmt.ColumnFloat(12)
	mediaData.Iso = stmt.ColumnFloat(13)
	mediaData.Lens = stmt.ColumnText(14)
	mediaData.Camera = stmt.ColumnText(15)
	mediaData.Focallength = stmt.ColumnFloat(16)
	mediaData.Altitude = stmt.ColumnFloat(17)
	mediaData.Latitude = stmt.ColumnFloat(18)
	mediaData.Longitude = stmt.ColumnFloat(19)
	mediaData.Mediatype = stmt.ColumnText(20)
	mediaData.Focusdistance = stmt.ColumnFloat(21)
	mediaData.Focallength35 = stmt.ColumnFloat(22)
	mediaData.Color = stmt.ColumnText(23)
	mediaData.Location = stmt.ColumnText(24)
	mediaData.Description = stmt.ColumnText(25)
	mediaData.Title = stmt.ColumnText(26)
	mediaData.Software = stmt.ColumnText(27)
	mediaData.Offset = float64(stmt.ColumnFloat(28))
	mediaData.Rotation = float64(stmt.ColumnFloat(29))

	item, err := parseMediaRow(mediaData)
	if err != nil {
		return Media{}, fmt.Errorf("error parsing media item: %v", err)
	}

	item.Srcset = sizes.Srcset(item.Hash, item.Width, item.Path, c)

	err = stmt.Finalize()
	if err != nil {
		return Media{}, fmt.Errorf("error finalizing statement: %v", err)
	}

	return item, nil
}
