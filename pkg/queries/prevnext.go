package queries

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/sizes"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// GetNext returns the media items after the current media item in chronological order.
func GetNext(date time.Time, hash uint32, total int, params FilterParams, previous []PrevNext, c Conf) ([]PrevNext, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer pool.Close()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	table := "media"
	term, err := sanitizeSearchInput(params.Term)
	if err != nil {
		c.Logger.Error("error formatting term query", "error", err)
	}
	if term != "" {
		table = `SELECT * FROM images_virtual(?)`
	}

	camera := params.Camera
	if camera != "" {
		camera = `AND i.camera =? `
	}

	lens := params.Lens
	if lens != "" {
		var p string
		for k, v := range c.Aliases.Lenses {
			if k == params.Lens {
				for _, value := range c.Aliases.Lenses {
					if v == value {
						p = fmt.Sprintf("%s%s", p, "?,")
					}
				}
			}
			if v == params.Lens {
				p = fmt.Sprintf("%s%s", p, "?,")
			}
		}

		if p != "" {
			lens = fmt.Sprintf(`AND i.lens in (%s)`, strings.TrimSuffix(p, ","))
		} else {
			lens = `AND i.lens =? `
		}
	}

	mediatype := params.MediaType
	if mediatype != "" {
		mediatype = `AND i.mediatype =? `
	}

	software := params.Software
	if software != "" {
		software = `AND i.software =? `
	}

	f35 := ""
	focallength35 := params.FocalLength35
	if focallength35 != 0 {
		f35 = `AND i.focallength35 =? `
	}

	folder := ""
	if params.Folder != "" {
		folder = `AND folder =? `
	}

	firstJoin := ""
	secondJoin := ""
	if params.Subject != "" {
		firstJoin =
			`JOIN
				images_tags i_t ON i.hash = i_t.image_id
			JOIN
				tags t ON i_t.tag_id = t.id`
		secondJoin = `AND t.key =?`
	} else if params.Folder != "" {
		firstJoin =
			`JOIN
			folders f ON f.key = i.folder`
		secondJoin = `AND f.key =?`
	}

	// Build an array of previous ids to exclude in case media items have the same datetime.
	var previous_ids []string
	for _, p := range previous {
		previous_ids = append(previous_ids, fmt.Sprint(p.Hash))
	}

	query := fmt.Sprintf(
		`SELECT DISTINCT
			i.hash,
			i.color,
			i.mediatype,
			i.path,
			i.width,
			i.height,
			i.date,
			i.offset
		FROM (%s) i
		%s
		WHERE i.hash !=?
		AND i.hash NOT IN (%s)
		AND i.rating >=?
		AND i.date <=?
		AND i.date !=?
		AND i.date !=?
		%s
		%s
		%s
		%s
		%s
		%s
		%s
		GROUP BY i.date
		ORDER BY i.date desc LIMIT %d`, table, firstJoin, strings.Join(previous_ids[:], ","), secondJoin, folder, camera, lens, mediatype, software, f35, total)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing SELECT statement: %v", err)
	}

	paramIdx := 1
	if term != "" {
		stmt.BindText(paramIdx, term)
		paramIdx++
	}
	stmt.BindInt64(paramIdx, int64(hash))
	paramIdx++
	stmt.BindInt64(paramIdx, int64(params.Rating))
	paramIdx++
	stmt.BindText(paramIdx, date.Format("2006-01-02T15:04:05.000Z"))
	paramIdx++
	stmt.BindText(paramIdx, date.Format("2006-01-02T15:04:05.000Z"))
	paramIdx++
	stmt.BindText(paramIdx, date.Format("0001-01-01T00:00:00.000Z"))
	paramIdx++

	if secondJoin != "" {
		if params.Subject != "" {
			stmt.BindText(paramIdx, params.Subject)
		} else if params.Folder != "" {
			stmt.BindText(paramIdx, params.Folder)
		}
		paramIdx++
	}

	if params.Folder != "" {
		stmt.BindText(paramIdx, params.Folder)
		paramIdx++
	}

	if camera != "" {
		stmt.BindText(paramIdx, params.Camera)
		paramIdx++
	}
	if lens != "" {
		var exists bool
		for k, v := range c.Aliases.Lenses {
			if k == params.Lens || v == params.Lens {
				exists = true
			}
		}
		if exists {
			for k, v := range c.Aliases.Lenses {
				if k == params.Lens {
					for key, value := range c.Aliases.Lenses {
						if v == value {
							stmt.BindText(paramIdx, key)
							paramIdx++
						}
					}

				}
				if v == params.Lens {
					stmt.BindText(paramIdx, k)
					paramIdx++
				}
			}
		} else {
			stmt.BindText(paramIdx, params.Lens)
			paramIdx++
		}
	}
	if mediatype != "" {
		stmt.BindText(paramIdx, params.MediaType)
		paramIdx++
	}
	if software != "" {
		stmt.BindText(paramIdx, params.Software)
		paramIdx++
	}
	if focallength35 != 0 {
		stmt.BindFloat(paramIdx, focallength35)
		paramIdx++
	}

	next := make([]PrevNext, 0)
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("error stepping through result set: %v", err)
		}
		if !hasRow {
			break
		}

		media := PrevNext{
			Hash:      uint32(stmt.ColumnInt64(0)),
			Color:     stmt.ColumnText(1),
			Mediatype: stmt.ColumnText(2),
			Path:      stmt.ColumnText(3),
			Width:     int(stmt.ColumnInt64(4)),
			Height:    int(stmt.ColumnInt64(5)),
		}

		t, err := time.Parse("2006-01-02T15:04:05.000Z", stmt.ColumnText(6))
		if err != nil {
			return nil, fmt.Errorf("error parsing date for media item: %v", err)
		}

		media.Date = t
		media.Srcset = sizes.Srcset(media.Hash, media.Width, media.Path, c)

		next = append(next, media)
	}

	return next, nil
}

// GetPrevious returns the media items before the current media item in chronological order.
func GetPrevious(date time.Time, hash uint32, params FilterParams, c Conf) ([]PrevNext, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer pool.Close()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	table := "media"
	term, err := sanitizeSearchInput(params.Term)
	if err != nil {
		c.Logger.Error("error formatting term query", "error", err)
	}
	if term != "" {
		table = `SELECT * FROM images_virtual(?)`
	}

	camera := params.Camera
	if camera != "" {
		camera = `AND i.camera =? `
	}

	lens := params.Lens
	if lens != "" {
		var p string
		for k, v := range c.Aliases.Lenses {
			if k == params.Lens {
				for _, value := range c.Aliases.Lenses {
					if v == value {
						p = fmt.Sprintf("%s%s", p, "?,")
					}
				}
			}
			if v == params.Lens {
				p = fmt.Sprintf("%s%s", p, "?,")
			}
		}

		if p != "" {
			lens = fmt.Sprintf(`AND i.lens in (%s)`, strings.TrimSuffix(p, ","))
		} else {
			lens = `AND i.lens =? `
		}
	}

	mediatype := params.MediaType
	if mediatype != "" {
		mediatype = `AND i.mediatype =? `
	}

	software := params.Software
	if software != "" {
		software = `AND i.software =? `
	}

	f35 := ""
	focallength35 := params.FocalLength35
	if focallength35 != 0 {
		f35 = `AND i.focallength35 =? `
	}

	folder := ""
	if params.Folder != "" {
		folder = `AND folder =? `
	}

	firstJoin := ""
	secondJoin := ""
	if params.Subject != "" {
		firstJoin =
			`JOIN
				images_tags i_t ON i.hash = i_t.image_id
			JOIN
				tags t ON i_t.tag_id = t.id`
		secondJoin = `AND t.key =?`
	} else if params.Folder != "" {
		firstJoin =
			`JOIN
			folders f ON f.key = i.folder`
		secondJoin = `AND f.key =?`
	}

	query := fmt.Sprintf(
		`SELECT hash, color, mediatype, path, width, height, date FROM
			(SELECT DISTINCT
				i.hash,
				i.color,
				i.mediatype,
				i.path,
				i.width,
				i.height,
				i.date
			FROM
				(%s) i
			%s
			WHERE
				i.hash !=?
			%s
			AND
				i.rating >=?
			AND
				i.date >=?
			AND
				i.date !=?
			AND
				i.date !=?
			%s
			%s
			%s
			%s
			%s
			%s
			GROUP BY i.date
			ORDER BY i.date ASC LIMIT 3)
		GROUP BY date
		ORDER BY date DESC`,
		table, firstJoin, secondJoin, folder, camera, lens, mediatype, software, f35)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing SELECT statement: %v", err)
	}

	paramIdx := 1
	if term != "" {
		stmt.BindText(paramIdx, term)
		paramIdx++
	}

	stmt.BindInt64(paramIdx, int64(hash))

	paramIdx++
	if secondJoin != "" {
		if params.Subject != "" {
			stmt.BindText(paramIdx, params.Subject)
		} else if params.Folder != "" {
			stmt.BindText(paramIdx, params.Folder)
		}
		paramIdx++
	}

	stmt.BindInt64(paramIdx, int64(params.Rating))
	paramIdx++
	stmt.BindText(paramIdx, date.Format("2006-01-02T15:04:05.000Z"))
	paramIdx++
	stmt.BindText(paramIdx, date.Format("2006-01-02T15:04:05.000Z"))
	paramIdx++
	stmt.BindText(paramIdx, date.Format("0001-01-01T00:00:00.000Z"))
	paramIdx++

	if params.Folder != "" {
		stmt.BindText(paramIdx, params.Folder)
		paramIdx++
	}

	if camera != "" {
		stmt.BindText(paramIdx, params.Camera)
		paramIdx++
	}
	if lens != "" {
		var exists bool
		for k, v := range c.Aliases.Lenses {
			if k == params.Lens || v == params.Lens {
				exists = true
			}
		}
		if exists {
			for k, v := range c.Aliases.Lenses {
				if k == params.Lens {
					for key, value := range c.Aliases.Lenses {
						if v == value {
							stmt.BindText(paramIdx, key)
							paramIdx++
						}
					}

				}
				if v == params.Lens {
					stmt.BindText(paramIdx, k)
					paramIdx++
				}
			}
		} else {
			stmt.BindText(paramIdx, params.Lens)
			paramIdx++
		}
	}
	if mediatype != "" {
		stmt.BindText(paramIdx, params.MediaType)
		paramIdx++
	}
	if software != "" {
		stmt.BindText(paramIdx, params.Software)
		paramIdx++
	}
	if focallength35 != 0 {
		stmt.BindFloat(paramIdx, focallength35)
		paramIdx++
	}

	previous := make([]PrevNext, 0)
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("error stepping through result set: %v", err)
		}
		if !hasRow {
			break
		}

		media := PrevNext{
			Hash:      uint32(stmt.ColumnInt64(0)),
			Color:     stmt.ColumnText(1),
			Mediatype: stmt.ColumnText(2),
			Path:      stmt.ColumnText(3),
			Width:     int(stmt.ColumnInt64(4)),
			Height:    int(stmt.ColumnInt64(5)),
		}

		t, err := time.Parse("2006-01-02T15:04:05.000Z", stmt.ColumnText(6))
		if err != nil {
			return nil, fmt.Errorf("error parsing date for media item: %v", err)
		}

		media.Date = t
		media.Srcset = sizes.Srcset(media.Hash, media.Width, media.Path, c)

		previous = append(previous, media)
	}

	return previous, nil
}
