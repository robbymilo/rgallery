package config

import (
	"html/template"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

type Meta = types.Meta
type Media = types.Media
type Conf = types.Conf

func CachePath(c Conf) string {
	cache_path, err := filepath.Abs(c.Cache)
	if err != nil {
		c.Logger.Error("error parsing cache dir", "error", err)
	}

	return cache_path
}

func MediaPath(c Conf) string {
	media_path, err := filepath.Abs(c.Media)
	if err != nil {
		c.Logger.Error("error parsing media dir", "error", err)
	}

	return media_path
}

// GetConf returns a Conf struct from the config file, cli flags, and env vars.
func GetConf(cCtx cli.Context, Commit, Tag string) Conf {
	var c Conf

	c.Logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	configPath := cCtx.String("config")

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			c.Logger.Error("Failed to read config file", "error", err)
		} else {

			err := yaml.Unmarshal(data, &c)
			if err != nil {
				c.Logger.Error("Error parsing YAML file", "error", err)
				os.Exit(1)
				return c
			}

			c.Logger.Info("Parsed config file at " + configPath)
		}
	}

	c.Cache = cCtx.String("cache")
	c.Data = cCtx.String("data")
	c.Dev = cCtx.Bool("dev")
	c.DisableAuth = cCtx.Bool("disable-auth")
	c.LocationService = cCtx.String("location-service")
	c.LocationDataset = cCtx.String("location-dataset")
	c.IncludeOriginals = cCtx.Bool("include-originals")
	c.Media = cCtx.String("media")
	c.OnThisDay = cCtx.Bool("on-this-day")
	c.PreGenerateThumb = cCtx.Bool("pregenerate-thumbs")
	c.Quality = cCtx.Int("quality")
	c.ResizeService = cCtx.String("resize_service")
	c.SessionLength = cCtx.Int("session-length")
	c.TileServer = cCtx.String("tile-server")

	c.Meta = Meta{
		Commit:     Commit,
		CustomHTML: template.HTML(c.CustomHTML),
		Tag:        Tag,
	}

	return c
}
