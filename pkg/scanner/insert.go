package scanner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	exiftool "github.com/barasher/go-exiftool"
	"github.com/cenkalti/dominantcolor"
	"github.com/disintegration/imageorient"
	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/exif"
	"github.com/robbymilo/rgallery/pkg/geo"
	"github.com/robbymilo/rgallery/pkg/hash"
	"github.com/robbymilo/rgallery/pkg/middleware"
	"github.com/robbymilo/rgallery/pkg/resize"
	"github.com/robbymilo/rgallery/pkg/transcode"
	"github.com/robbymilo/rgallery/pkg/types"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Subject = types.Subject

// addImage gathers the exif data of an image and passes it to the db inserter.
func addImage(relative_path, absolute_path string, isUpdate, regenThumb bool, et *exiftool.Exiftool, h *geo.Handlers, c Conf, cache *cache.Cache) error {

	image, _, err := exif.GetImageExif("image", relative_path, absolute_path, et, h, c)
	if err != nil {
		return fmt.Errorf("error getting exif: %v", err)
	}

	generated, err := resize.HandleResize(regenThumb, image, c)
	if err != nil {
		return fmt.Errorf("error resizing image: %v", err)
	}

	err = insertMediaItem(image, c)
	if err != nil {
		return fmt.Errorf("error inserting image: %v", err)
	}

	if isUpdate {
		c.Logger.Info("updated image with " + strconv.Itoa(generated) + " thumbnails: " + relative_path)
	} else {
		c.Logger.Info("added image with " + strconv.Itoa(generated) + " thumbnails: " + relative_path)
	}

	cache.Flush()
	middleware.RemoveEtags()

	return nil

}

// addVideo gathers the exif data of an image and passes it to the db inserter.
func addVideo(relative_path, absolute_path string, isUpdate, regenThumb bool, et *exiftool.Exiftool, h *geo.Handlers, c Conf, cache *cache.Cache) error {

	media, _, err := exif.GetImageExif("video", relative_path, absolute_path, et, h, c)
	if err != nil {
		return fmt.Errorf("error getting video exif: %v", err)
	}

	generated, err := resize.HandleResize(regenThumb, media, c)
	if err != nil {
		return fmt.Errorf("error resizing video thumb: %v", err)
	}

	im, err := resize.GenerateSingleThumb(absolute_path, media, 400, c)
	if err != nil {
		return fmt.Errorf("error creating video thumb for dominant color: %v ", err)
	}

	img, _, err := imageorient.Decode(bytes.NewReader(im))
	if err != nil {
		return fmt.Errorf("error decoding video thumb for dominant color: %v", err)
	}

	// dominant color
	color := dominantcolor.Hex(dominantcolor.Find(img))
	media.Color = color

	// pre-transcode image
	if c.PreGenerateThumb || regenThumb {
		index_file := transcode.CreateHLSIndexFilePath(media.Hash, c)
		err = transcode.TranscodeVideo(absolute_path, index_file, media.Hash, c.TranscodeResolution)
		if err != nil {
			return fmt.Errorf("error transcoding video: %v", err)
		}
	}

	err = insertMediaItem(media, c)
	if err != nil {
		fmt.Printf("error inserting image: %s %s\n", media.Path, err)
		return err
	}

	cache.Flush()
	middleware.RemoveEtags()

	if isUpdate {
		c.Logger.Info("updated video with " + strconv.Itoa(generated) + " thumbnails: " + relative_path)
	} else {
		c.Logger.Info("added video with " + strconv.Itoa(generated) + " thumbnails: " + relative_path)
	}

	return nil

}

// insertMediaItem coordinates inserting a media item, and it's tags, folders and their relationships to the database.
func insertMediaItem(media Media, c Conf) error {

	if media.Date.Year() == 0001 {
		return errors.New("skipping insert, media has no date")
	}

	// Create a connection pool
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadWrite,
		PoolSize: 1,
	})
	if err != nil {
		return fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer pool.Close()

	// Take a connection from the pool
	conn, err := pool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	// Set busy timeout to handle database locks
	err = sqlitex.ExecuteTransient(conn, "PRAGMA busy_timeout = 5000", nil)
	if err != nil {
		return fmt.Errorf("error setting busy timeout: %v", err)
	}

	// Retry logic for transaction
	maxRetries := 3
	var retryCount int

	for retryCount < maxRetries {
		err = sqlitex.Execute(conn, "BEGIN TRANSACTION", nil)
		if err != nil {
			retryCount++
			if retryCount == maxRetries {
				return fmt.Errorf("failed to begin transaction after %d attempts: %v", maxRetries, err)
			}
			c.Logger.Warn("transaction begin failed, retrying", "error", err, "attempt", retryCount)
			continue
		}

		folder_id := hash.GetHash(media.Folder)

		// check if folder exists
		var folderExists bool
		err = sqlitex.ExecuteTransient(conn, "SELECT id FROM folders WHERE id = ?", &sqlitex.ExecOptions{
			Args: []interface{}{folder_id},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				folderExists = true
				return nil
			},
		})
		if err != nil {
			rollbackErr := sqlitex.Execute(conn, "ROLLBACK", nil)
			if rollbackErr != nil {
				c.Logger.Error("error executing rollback", "error", rollbackErr)
			}
			retryCount++
			if retryCount == maxRetries {
				return fmt.Errorf("failed to query folders after %d attempts: %v", maxRetries, err)
			}
			c.Logger.Warn("folder query failed, retrying", "error", err, "attempt", retryCount)
			continue
		}

		// add folder if it does not exist
		if !folderExists {
			err = sqlitex.ExecuteTransient(conn, "INSERT INTO folders(id, key) VALUES (?, ?)", &sqlitex.ExecOptions{
				Args: []interface{}{folder_id, media.Folder},
			})
			if err != nil {
				rollbackErr := sqlitex.Execute(conn, "ROLLBACK", nil)
				if rollbackErr != nil {
					c.Logger.Error("error executing rollback", "error", rollbackErr)
				}
				return fmt.Errorf("error inserting folder: %v", err)
			}

			c.Logger.Info("added folder " + media.Folder)
		}

		// handle tag inserts
		for _, tag := range media.Subject {
			tag_id := hash.GetHash(tag.Key)

			// check if tag exists
			var tagExists bool
			err = sqlitex.ExecuteTransient(conn, "SELECT id FROM tags WHERE id = ?", &sqlitex.ExecOptions{
				Args: []interface{}{tag_id},
				ResultFunc: func(stmt *sqlite.Stmt) error {
					tagExists = true
					return nil
				},
			})
			if err != nil {
				rollbackErr := sqlitex.Execute(conn, "ROLLBACK", nil)
				if rollbackErr != nil {
					c.Logger.Error("error executing rollback", "error", rollbackErr)
				}
				return fmt.Errorf("error checking tag existence: %v", err)
			}

			// add tag if it does not exist
			if !tagExists {
				err = sqlitex.ExecuteTransient(conn, "INSERT INTO tags(id, key, value) VALUES (?, ?, ?)", &sqlitex.ExecOptions{
					Args: []interface{}{tag_id, tag.Key, tag.Value},
				})
				if err != nil {
					rollbackErr := sqlitex.Execute(conn, "ROLLBACK", nil)
					if rollbackErr != nil {
						c.Logger.Error("error executing rollback", "error", rollbackErr)
					}
					return fmt.Errorf("error inserting tag: %v", err)
				}

				c.Logger.Info("added tag " + tag.Value)
			}

			// add tag-image many-to-many relationship
			err = sqlitex.ExecuteTransient(conn, "INSERT INTO images_tags(image_id, tag_id) VALUES (?, ?)", &sqlitex.ExecOptions{
				Args: []interface{}{media.Hash, tag_id},
			})
			if err != nil {
				rollbackErr := sqlitex.Execute(conn, "ROLLBACK", nil)
				if rollbackErr != nil {
					c.Logger.Error("error executing rollback", "error", rollbackErr)
				}
				return fmt.Errorf("error inserting image-tag relationship: %v", err)
			}
		}

		// insert image
		s, err := json.Marshal(media.Subject)
		if err != nil {
			rollbackErr := sqlitex.Execute(conn, "ROLLBACK", nil)
			if rollbackErr != nil {
				c.Logger.Error("error executing rollback", "error", rollbackErr)
			}
			return fmt.Errorf("error marshaling subject: %v", err)
		}

		query := fmt.Sprintf("INSERT INTO media(%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", database.Columns())
		err = sqlitex.ExecuteTransient(conn, query, &sqlitex.ExecOptions{
			Args: []interface{}{
				media.Hash,
				media.Path,
				string(s),
				media.Width,
				media.Height,
				media.Ratio,
				media.Padding,
				string(media.Date.Format("2006-01-02T15:04:05.000Z")),
				string(media.Modified.Format("2006-01-02T15:04:05.000Z")),
				media.Folder,
				media.Rating,
				media.ShutterSpeed,
				media.Aperture,
				media.Iso,
				media.Lens,
				media.Camera,
				media.Focallength,
				media.Altitude,
				media.Latitude,
				media.Longitude,
				media.Type,
				media.FocusDistance,
				media.FocalLength35,
				media.Color,
				media.Location,
				media.Description,
				media.Title,
				media.Software,
				media.Offset,
				media.Rotation,
			},
		})
		if err != nil {
			rollbackErr := sqlitex.Execute(conn, "ROLLBACK", nil)
			if rollbackErr != nil {
				c.Logger.Error("error executing rollback", "error", rollbackErr)
			}
			return fmt.Errorf("error inserting image: %v", err)
		}

		// Try to commit the transaction
		err = sqlitex.Execute(conn, "COMMIT", nil)
		if err != nil {
			// If we get a database locked error, retry
			if err.Error() == "database is locked" {
				retryCount++
				if retryCount == maxRetries {
					return fmt.Errorf("failed to commit transaction after %d attempts: %v", maxRetries, err)
				}
				c.Logger.Warn("transaction commit failed due to database lock, retrying", "attempt", retryCount)
				continue
			}
			return fmt.Errorf("error committing transaction: %v", err)
		}

		// If we get here, the transaction was successful
		break
	}

	return nil
}
