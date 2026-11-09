package queries

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/robbymilo/rgallery/pkg/sizes"
	"zombiezen.com/go/sqlite"
)

// parseMediaRows takes SQL rows and returns a list of media items.
func parseMediaRows(stmt *sqlite.Stmt, c Conf) ([]Media, error) {
	var result []Media
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("error stepping through result set: %v", err)
		}
		if !hasRow {
			break
		}

		media := Media{
			Hash:          uint32(stmt.ColumnInt64(0)),
			Path:          stmt.ColumnText(1),
			Width:         stmt.ColumnInt(3),
			Height:        stmt.ColumnInt(4),
			Ratio:         float32(stmt.ColumnFloat(5)),
			Padding:       float32(stmt.ColumnFloat(6)),
			Folder:        stmt.ColumnText(9),
			Rating:        stmt.ColumnFloat(10),
			ShutterSpeed:  stmt.ColumnText(11),
			Aperture:      stmt.ColumnFloat(12),
			Iso:           stmt.ColumnFloat(13),
			Lens:          stmt.ColumnText(14),
			Camera:        stmt.ColumnText(15),
			Focallength:   stmt.ColumnFloat(16),
			Altitude:      stmt.ColumnFloat(17),
			Latitude:      stmt.ColumnFloat(18),
			Longitude:     stmt.ColumnFloat(19),
			Type:          stmt.ColumnText(20),
			FocusDistance: stmt.ColumnFloat(21),
			FocalLength35: stmt.ColumnFloat(22),
			Color:         stmt.ColumnText(23),
			Location:      stmt.ColumnText(24),
			Description:   stmt.ColumnText(25),
			Title:         stmt.ColumnText(26),
			Software:      stmt.ColumnText(27),
			Offset:        stmt.ColumnFloat(28),
			Rotation:      stmt.ColumnFloat(29),
		}

		subjectsJSON := make([]Subject, 0)
		err = json.Unmarshal([]byte(stmt.ColumnText(2)), &subjectsJSON)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling json for media item: %v", err)
		}
		media.Subject = subjectsJSON

		dateStr := stmt.ColumnText(7)
		date, err := time.Parse("2006-01-02T15:04:05.000Z", dateStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing date column: %v", err)
		}
		media.Date = date

		modifiedStr := stmt.ColumnText(8)
		modified, err := time.Parse(time.RFC3339Nano, modifiedStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing modified column: %v", err)
		}
		media.Modified = modified

		media.Srcset = sizes.Srcset(media.Hash, media.Width, media.Path, c)

		result = append(result, media)
	}

	return result, nil
}

// parseMediaRow takes a single SQL row and returns a media item.
func parseMediaRow(r DatabaseMedia) (Media, error) {
	var t time.Time
	t, err := time.Parse("2006-01-02T15:04:05.000Z", r.Date)
	if err != nil {
		return Media{}, fmt.Errorf("error parsing date for media item: %v", err)
	}

	m, err := time.Parse(time.RFC3339Nano, r.Modified)
	if err != nil {
		return Media{}, fmt.Errorf("error parsing modified date for media item: %v", err)
	}

	s := make([]Subject, 0)
	err = json.Unmarshal([]byte(r.Subject), &s)
	if err != nil {
		return Media{}, fmt.Errorf("error unmarshalling json for media item: %v", err)
	}

	media := Media{
		Hash:          r.Hash,
		Path:          r.Path,
		Subject:       s,
		Width:         r.Width,
		Height:        r.Height,
		Ratio:         r.Ratio,
		Padding:       r.Padding,
		Date:          t,
		Modified:      m,
		Folder:        r.Folder,
		Rating:        r.Rating,
		ShutterSpeed:  r.ShutterSpeed,
		Aperture:      r.Aperture,
		Iso:           r.Iso,
		Lens:          r.Lens,
		Camera:        r.Camera,
		Focallength:   r.Focallength,
		Altitude:      r.Altitude,
		Latitude:      r.Latitude,
		Longitude:     r.Longitude,
		Type:          r.Mediatype,
		FocusDistance: r.Focusdistance,
		FocalLength35: r.Focallength35,
		Color:         r.Color,
		Location:      r.Location,
		Description:   r.Description,
		Title:         r.Title,
		Software:      r.Software,
		Offset:        r.Offset,
		Rotation:      r.Rotation,
	}

	return media, nil
}
