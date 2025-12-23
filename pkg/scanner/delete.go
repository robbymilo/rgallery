package scanner

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/config"
	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/hash"
	"github.com/robbymilo/rgallery/pkg/middleware"
	"github.com/robbymilo/rgallery/pkg/sizes"
)

// deleteMediaItem coordinates the removal of a media item, any associated tags and folders, and thumbnails.
func deleteMediaItem(path string, removeDeletedThumbnails bool, media Media, c Conf, cache *cache.Cache) error {

	cache.Flush()
	middleware.RemoveEtags()

	// check if image exists in db
	if media.Path == path {

		db, err := sql.Open("sqlite", database.NewConnectionString(c))
		if err != nil {
			return err
		}
		defer db.Close()

		// delete image
		_, err = db.Exec("DELETE FROM media WHERE hash =?", media.Hash)
		if err != nil {
			return err
		}

		// delete tag relationships
		_, err = db.Exec("DELETE FROM images_tags WHERE image_id =?", media.Hash)
		if err != nil {
			return err
		}

		// delete tag relationships
		for _, tag := range media.Subject {
			// check if other items have the tag
			query := `SELECT t.id FROM tags t JOIN images_tags i_a ON t.id = i_a.tag_id WHERE i_a.tag_id = ?`
			rows, err := db.Query(query, hash.GetHash(tag.Key))
			if err != nil {
				return err
			}
			defer rows.Close()

			// if no other items have the tag
			if !rows.Next() {
				// delete tag
				c.Logger.Info("deleting tag" + tag.Key)
				_, err = db.Exec("DELETE FROM tags WHERE id =?", hash.GetHash(tag.Key))
				if err != nil {
					return err
				}

			}

		}

		// delete folder
		// check if other items have the folder
		query := `SELECT f.key FROM folders f JOIN media i ON f.key = i.folder WHERE i.folder = ?`
		rows, err := db.Query(query, media.Folder)
		if err != nil {
			return err
		}
		defer rows.Close()

		// if no other items have the folder
		if !rows.Next() {
			// delete folder
			c.Logger.Info("deleting folder" + media.Folder)
			_, err = db.Exec("DELETE FROM folders WHERE key =?", media.Folder)
			if err != nil {
				return err
			}

		}

		if removeDeletedThumbnails {
			err = removeThumbs(path, media, c)
		}

		return err

	} else {
		return errors.New("media path in db does not equal supplied path")
	}

}

func removeThumbs(path string, media Media, c Conf) error {
	var err error
	// remove thumbnails
	file := fmt.Sprintf("%d.jpg", media.Hash)
	c.Logger.Info("removing thumbnails for " + path)

	var thumb_file string
	cache_path := config.CachePath(c)
	final := false

	for _, size := range sizes.GetSizes() {

		if size <= media.Width {
			thumb_file = filepath.Join(cache_path, strconv.Itoa(size), file)
			if _, err := os.Stat(thumb_file); err == nil {
				// remove file
				err := os.Remove(thumb_file)
				if err != nil {
					return fmt.Errorf("error removing thumbnail file: %v", err)
				}

			}
		} else if !final {
			final = true
			thumb_file = filepath.Join(cache_path, strconv.Itoa(media.Width), file)
			if _, err := os.Stat(thumb_file); err == nil {
				// remove file
				err := os.Remove(thumb_file)
				if err != nil {
					return fmt.Errorf("error removing thumbnail file: %v", err)
				}

			}
		}
	}

	return err
}
