package scanner

import (
	"context"
	"path/filepath"

	exiftool "github.com/barasher/go-exiftool"
	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/config"
	"github.com/robbymilo/rgallery/pkg/geo"
)

// updateMediaItem removes a media item from the db, and initiates a new addition of the same item.
func updateMediaItem(ctx context.Context, relative_path string, regenThumb bool, et, convertedEt *exiftool.Exiftool, h *geo.Handlers, c Conf, media Media, cache *cache.Cache, metrics *scanMetrics, signature fileSignature) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// remove media
	err := deleteMediaItem(relative_path, false, media, c, cache)
	if err != nil {
		c.Logger.Error("error deleting media item", "err", err)
		return err
	}

	absolute_path := filepath.Join(config.MediaPath(c), relative_path)

	if isImage(absolute_path) {
		err = addImage(ctx, relative_path, absolute_path, signature, true, regenThumb, et, convertedEt, h, c, cache, metrics)
		if err != nil {
			c.Logger.Error("error updating image", "err", err)
			return err
		}
	}

	if isVideo(absolute_path) {
		err = addVideo(ctx, relative_path, absolute_path, signature, true, regenThumb, et, convertedEt, h, c, cache, metrics)
		if err != nil {
			c.Logger.Error("error updating video", "err", err)
			return err
		}
	}

	return nil

}
