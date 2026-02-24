package scanner

import (
	"path/filepath"

	exiftool "github.com/barasher/go-exiftool"
	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/config"
	"github.com/robbymilo/rgallery/pkg/geo"
)

// updateMediaItem removes a media item from the db, and initiates a new addition of the same item.
func updateMediaItem(relative_path string, regenThumb bool, et *exiftool.Exiftool, h *geo.Handlers, c Conf, media Media, cache *cache.Cache) error {

	// remove media
	err := deleteMediaItem(relative_path, false, media, c, cache)
	if err != nil {
		c.Logger.Error("error deleting media item", "err", err)
		return err
	}

	absolute_path := filepath.Join(config.MediaPath(c), relative_path)

	if isImage(absolute_path) {
		err = addImage(relative_path, absolute_path, true, regenThumb, et, h, c, cache)
		if err != nil {
			c.Logger.Error("error updating image", "err", err)
			return err
		}
	}

	if isVideo(absolute_path) {
		err = addVideo(relative_path, absolute_path, true, regenThumb, et, h, c, cache)
		if err != nil {
			c.Logger.Error("error updating video", "err", err)
			return err
		}
	}

	return nil

}
