package queries

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/types"
)

type RawMinimalMedia = types.RawMinimalMedia
type Segment = types.Segment
type SegmentMedia = types.SegmentMedia
type SegmentGroup = types.SegmentGroup
type ResponseSegment = types.ResponseSegment
type GearItem = types.GearItem
type GearItems = types.GearItems
type Conf = types.Conf
type PrevNext = types.PrevNext

type TimelineResponse struct {
	Meta     Meta           `json:"meta"`
	Timeline []TimelineItem `json:"timeline"`
	Photos   []Photo        `json:"photos"`
}

type Meta struct {
	Total      int    `json:"total"`
	PageSize   int    `json:"pagesize"`
	NextCursor string `json:"nextCursor"`
}

type TimelineItem struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type Photo struct {
	Id uint32 `json:"id"`
	W  int    `json:"w"`
	H  int    `json:"h"`
	C  string `json:"c"`
	T  string `json:"t,omitempty"`
	D  string `json:"d"` // YYYY-MM-DD
}

// GetTimeline returns media items in the new cursor-based format.
func GetTimeline(params *FilterParams, c Conf) (*TimelineResponse, error) {
	// Safety: Default PageSize
	if params.PageSize <= 0 {
		params.PageSize = 1000
	}

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

	response := &TimelineResponse{
		Timeline: []TimelineItem{},
		Photos:   []Photo{},
	}
	response.Meta.PageSize = params.PageSize

	total, err := fetchTotalCount(conn, params, c)
	if err != nil {
		return nil, err
	}
	response.Meta.Total = total

	// calculate histogram for all items when cursor is 0
	if params.Cursor == 0 {
		timeline, err := fetchTimelineStats(conn, params, c)
		if err != nil {
			return nil, err
		}
		if timeline != nil {
			response.Timeline = timeline
		}
	}

	photos, err := fetchPhotos(conn, params, c)
	if err != nil {
		return nil, err
	}
	response.Photos = photos

	// 4. Next Cursor Logic
	if len(photos) >= params.PageSize {
		next := params.Cursor + params.PageSize
		if next < total {
			response.Meta.NextCursor = strconv.Itoa(next)
		}
	}

	return response, nil
}

func fetchTotalCount(conn *sqlite.Conn, params *FilterParams, c Conf) (int, error) {
	baseQuery, args, err := buildBaseQuery(params, c)
	if err != nil {
		return 0, err
	}

	// Count unique timestamps
	query := fmt.Sprintf("SELECT COUNT(DISTINCT m.date) %s", baseQuery)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("prepare count: %w (query: %s)", err, query)
	}
	defer func() {
		if err := stmt.Finalize(); err != nil {
			c.Logger.Error("timeline: finalize stmt error", "err", err)
		}
	}()

	bindArgs(stmt, args)

	hasRow, err := stmt.Step()
	if err != nil {
		return 0, err
	}
	if hasRow {
		return int(stmt.ColumnInt64(0)), nil
	}
	return 0, nil
}

func fetchTimelineStats(conn *sqlite.Conn, params *FilterParams, c Conf) ([]TimelineItem, error) {
	baseQuery, args, err := buildBaseQuery(params, c)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT substr(m.date, 1, 10) as day, COUNT(DISTINCT m.date)
		%s
		GROUP BY day
		ORDER BY day DESC`, baseQuery)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("prepare timeline: %w", err)
	}
	defer func() {
		if err := stmt.Finalize(); err != nil {
			c.Logger.Error("timeline: finalize stmt error: %v", "err", err)
		}
	}()

	bindArgs(stmt, args)

	results := make([]TimelineItem, 0)
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, err
		}
		if !hasRow {
			break
		}
		results = append(results, TimelineItem{
			Date:  stmt.ColumnText(0),
			Count: int(stmt.ColumnInt64(1)),
		})
	}
	return results, nil
}

func fetchPhotos(conn *sqlite.Conn, params *FilterParams, c Conf) ([]Photo, error) {
	baseQuery, args, err := buildBaseQuery(params, c)
	if err != nil {
		return nil, err
	}

	// Fetch specific page, grouping by date to deduplicate exact timestamps.
	query := fmt.Sprintf(`
		SELECT DISTINCT
			m.hash,
			m.width,
			m.height,
			m.color,
			m.date,
			m.mediatype

		%s
		GROUP BY m.date
		ORDER BY m.%s %s
		LIMIT %d OFFSET %d`,
		baseQuery,
		params.OrderBy, params.Direction,
		params.PageSize, params.Cursor,
	)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("prepare photos: %w", err)
	}
	defer func() {
		if err := stmt.Finalize(); err != nil {
			c.Logger.Error("timeline: finalize stmt error: %v", "err", err)
		}
	}()

	bindArgs(stmt, args)

	results := make([]Photo, 0)
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, err
		}
		if !hasRow {
			break
		}

		rawDate := stmt.ColumnText(4)
		shortDate := rawDate
		if len(rawDate) >= 10 {
			shortDate = rawDate[:10]
		}

		photo := Photo{
			Id: uint32(stmt.ColumnInt64(0)),
			W:  int(stmt.ColumnInt64(1)),
			H:  int(stmt.ColumnInt64(2)),
			C:  stmt.ColumnText(3),
			D:  shortDate,
		}

		t := stmt.ColumnText(5)
		if t == "video" {
			photo.T = "video"
		}

		results = append(results, photo)

	}
	return results, nil
}

func buildBaseQuery(params *FilterParams, c Conf) (string, []interface{}, error) {
	var sb strings.Builder
	var args []interface{}
	var where []string

	term, _ := sanitizeSearchInput(params.Term)
	if term != "" {
		// join virtual table to main table
		sb.WriteString(" FROM media m ")
		sb.WriteString(" INNER JOIN images_virtual v ON m.hash = v.hash ")
		where = append(where, "images_virtual MATCH ?")
		args = append(args, term)
	} else {
		sb.WriteString(" FROM media m ")
	}

	if params.Subject != "" {
		sb.WriteString(" JOIN images_tags it ON m.hash = it.image_id ")
		sb.WriteString(" JOIN tags t ON it.tag_id = t.id ")
		where = append(where, "t.key = ?")
		args = append(args, params.Subject)
	}

	// Filters
	where = append(where, "m.rating >= ?")
	args = append(args, params.Rating)

	if params.From != "" {
		where = append(where, "m.date >= ?")
		args = append(args, params.From)
	}

	if params.To != "" {
		where = append(where, "m.date <= ?")
		args = append(args, params.To)
	}

	if params.Camera != "" {
		where = append(where, "m.camera = ?")
		args = append(args, params.Camera)
	}

	if params.Lens != "" {
		lensVariants := []string{params.Lens}
		for k, v := range c.Aliases.Lenses {
			if k == params.Lens {
				for ak, av := range c.Aliases.Lenses {
					if v == av {
						lensVariants = append(lensVariants, ak)
					}
				}
				lensVariants = append(lensVariants, v)
			} else if v == params.Lens {
				lensVariants = append(lensVariants, k)
			}
		}

		placeholders := make([]string, len(lensVariants))
		for i, v := range lensVariants {
			placeholders[i] = "?"
			args = append(args, v)
		}
		where = append(where, fmt.Sprintf("m.lens IN (%s)", strings.Join(placeholders, ",")))
	}

	if params.Folder != "" {
		where = append(where, "m.folder = ?")
		args = append(args, params.Folder)
	}

	if params.MediaType != "" {
		where = append(where, "m.mediatype = ?")
		args = append(args, params.MediaType)
	}

	if params.Software != "" {
		where = append(where, "m.software = ?")
		args = append(args, params.Software)
	}

	if params.FocalLength35 != 0 {
		where = append(where, "m.focallength35 = ?")
		args = append(args, params.FocalLength35)
	}

	if len(where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(where, " AND "))
	}

	return sb.String(), args, nil
}

func bindArgs(stmt *sqlite.Stmt, args []interface{}) {
	for i, arg := range args {
		idx := i + 1
		switch v := arg.(type) {
		case int:
			stmt.BindInt64(idx, int64(v))
		case int64:
			stmt.BindInt64(idx, v)
		case float64:
			stmt.BindFloat(idx, v)
		case string:
			stmt.BindText(idx, v)
		case bool:
			stmt.BindBool(idx, v)
		default:
			stmt.BindText(idx, fmt.Sprintf("%v", v))
		}
	}
}

func sanitizeSearchInput(input string) (string, error) {
	re, err := regexp.Compile(`[^\p{L}\p{N} ]+`)
	if err != nil {
		return "", err
	}
	cleaned := re.ReplaceAllString(input, "")
	return cleaned, nil
}
