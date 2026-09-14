package rgallery

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
	_ "time/tzdata"

	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/config"
	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/scanner"
	"github.com/robbymilo/rgallery/pkg/transcode"
	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/robbymilo/rgallery/pkg/users"
	cli "github.com/urfave/cli/v2"
	_ "modernc.org/sqlite"
)

type Conf = types.Conf
type Middleware func(http.HandlerFunc) http.HandlerFunc
type UserCredentials = types.UserCredentials

func validateVideoFlags(cCtx *cli.Context) error {
	if cCtx.Int("transcode-workers") < 1 || cCtx.Int("transcode-cache-mb") < 64 || cCtx.Int("transcode-resolution") < 2 {
		return fmt.Errorf("transcode-workers must be at least 1, transcode-cache-mb at least 64, and transcode-resolution at least 2")
	}
	return transcode.Validate(Conf{TranscodeResolution: cCtx.Int("transcode-resolution"), Transcode: config.VideoConf(*cCtx)})
}

func SetupApp(Commit, Tag string) {
	port := "3000"
	metricsPort := "3001"

	var tz string
	tz = os.Getenv("TZ")
	if tz == "" {
		tz = "UTC"
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Fatal("error getting timezone:", err)
	}
	time.Local = loc

	flags := []cli.Flag{
		&cli.BoolFlag{
			Name:  "dev",
			Usage: "Load assets from directory instead of embedding, allowing you to edit assets without a recompile. Disables caching of HTML and JSON responses.",
		},
		&cli.BoolFlag{
			Name:  "disable-auth",
			Usage: "Load rgallery without a login.",
		},
		&cli.StringFlag{
			Name:  "media",
			Usage: "Location of the media directory.",
			Value: "./media",
		},
		&cli.StringFlag{
			Name:  "data",
			Usage: "Location of the database directory",
			Value: "./data",
		},
		&cli.StringFlag{
			Name:  "cache",
			Usage: "Location of the cache directory for storing image thumbnails and video transcode files.",
			Value: "./cache",
		},
		&cli.StringFlag{
			Name:  "config",
			Usage: "Location of the config yaml file. Only needed if using lens aliases.",
			Value: "./config/config.yml",
		},
		&cli.IntFlag{
			Name:  "quality",
			Usage: "Thumbnail resize quality.",
			Value: 60,
		},
		&cli.IntFlag{
			Name:    "transcode-resolution",
			Usage:   "Maximum video long edge in pixels, without upscaling. 1280 allows 720p and 1920 allows 1080p for 16:9 sources. Individual quality profiles may use smaller dimensions.",
			Value:   1280,
			EnvVars: []string{"RGALLERY_TRANSCODE_RESOLUTION"},
		},
		&cli.StringFlag{Name: "transcode-quality", Value: "small", Usage: "Default video quality: saver, small, or high.", EnvVars: []string{"RGALLERY_TRANSCODE_QUALITY"}},
		&cli.StringFlag{Name: "transcode-mode", Value: "ondemand", Usage: "Video generation: ondemand, pregenerate (all profiles), or hybrid (default profile during scan). Independent of thumbnail generation.", EnvVars: []string{"RGALLERY_TRANSCODE_MODE"}},
		&cli.StringFlag{Name: "transcode-encoder", Value: "auto", Usage: "Video encoder: auto, cpu, or vaapi. Hardware failures fall back to CPU.", EnvVars: []string{"RGALLERY_TRANSCODE_ENCODER"}},
		&cli.StringFlag{Name: "transcode-device", Usage: "VA-API render device, e.g. /dev/dri/renderD128. Empty discovers accessible devices.", EnvVars: []string{"RGALLERY_TRANSCODE_DEVICE"}},
		&cli.IntFlag{Name: "transcode-crf", Value: -1, Usage: "CPU CRF for the default profile (0–51; higher means smaller files). -1 uses the profile default.", EnvVars: []string{"RGALLERY_TRANSCODE_CRF"}},
		&cli.StringFlag{Name: "transcode-preset", Value: "veryfast", Usage: "CPU encoder speed preset; slower presets require more processing time.", EnvVars: []string{"RGALLERY_TRANSCODE_PRESET"}},
		&cli.IntFlag{Name: "transcode-maxrate", Usage: "Video bitrate ceiling in kbps across all profiles. 0 uses each profile's ceiling.", EnvVars: []string{"RGALLERY_TRANSCODE_MAXRATE"}},
		&cli.IntFlag{Name: "transcode-audio-bitrate", Usage: "Audio bitrate in kbps (32–320). 0 uses each profile's default.", EnvVars: []string{"RGALLERY_TRANSCODE_AUDIO_BITRATE"}},
		&cli.IntFlag{Name: "transcode-workers", Value: 2, Usage: "Maximum concurrent video jobs (1–16).", EnvVars: []string{"RGALLERY_TRANSCODE_WORKERS"}},
		&cli.IntFlag{Name: "transcode-cache-mb", Value: 10240, Usage: "Video cache budget in MiB (minimum 64). Active output is protected during eviction.", EnvVars: []string{"RGALLERY_TRANSCODE_CACHE_MB"}},
		&cli.BoolFlag{
			Name:  "pregenerate-thumbs",
			Usage: "Generate image thumbnails, video posters, and short video previews during scan. Full playback encoding is controlled by transcode-mode.",
			Value: true,
		},
		&cli.StringFlag{
			Name:    "resize_service",
			Usage:   "URL for resize service.",
			EnvVars: []string{"RGALLERY_RESIZE_SERVICE"},
		},
		&cli.StringFlag{
			Name:    "location-service",
			Usage:   "URL for reverse geocode service.",
			EnvVars: []string{"RGALLERY_LOCATION_SERVICE"},
		},
		&cli.StringFlag{
			Name:  "location-dataset",
			Usage: "Dataset for reverse geocode lookup. Ex: Countries10, Countries110, Provinces10. Countries10 uses the least amount of memory, and Provinces10 the most.",
			Value: "Provinces10",
		},
		&cli.StringFlag{
			Name:    "tile-server",
			Usage:   "URL for GeoServer tiles in XYZ format, ex https://tile.thunderforest.com/cycle/{z}/{x}/{y}.png?apikey=your-api-key-here.",
			EnvVars: []string{"RGALLERY_TILE_SERVER"},
			Value:   "/api/tiles/{z}/{x}/{y}.png",
		},
		&cli.IntFlag{
			Name:    "session-length",
			Usage:   "Length of authenticated sessions in days.",
			EnvVars: []string{"RGALLERY_SESSION_LENGTH"},
			Value:   30,
		},
		&cli.BoolFlag{
			Name:  "include-originals",
			Usage: "Include original files in web view. Setting this to true may cause slower image loading performance.",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "memories",
			Usage: "Show media items that occurred on the current day in previous years.",
			Value: true,
		},
	}

	app := &cli.App{
		Name:        "rgallery",
		Usage:       "A photo and video application.",
		Description: "The timeline for your photo and video library.",
		Flags:       flags,
		Before:      validateVideoFlags,
		Action: func(cCtx *cli.Context) error {
			c := config.GetConf(*cCtx, Commit, Tag)
			transcode.For(c).LogConfiguration()

			c.Logger.Info("thumbnail dir located at " + config.CachePath(c))
			database.CreateDB(c)

			scanner.SetScanInProgress(false)

			// initialize cache
			cache := cache.New(-1, -1)

			// load metrics router
			m := SetupMetrics(c)
			if !c.DisableAuth {
				err := users.InitUser(c)
				if err != nil {
					log.Fatal(err.Error())
				}
			}

			go http.ListenAndServe(":"+metricsPort, m) //nolint:all

			// load router before starting scan as we need to check for geo and resize service
			r := SetupRouter(c, cache, Commit, Tag)

			// get total number of media items in db to determine if we need to scan on startup
			from := time.Unix(0, 0)
			to := time.Now()
			totalItems, err := queries.GetTotalMediaItems(0, from.Format(time.RFC3339), to.Format(time.RFC3339), "", "", c)
			if err != nil {
				log.Fatal("error getting total media items in db")
			}

			// scan on startup if no media items
			if totalItems == 0 {
				c.Logger.Info("no media items found in db, starting scan...")
				go scanner.BackgroundScan("default", c, cache) //nolint:all
			}

			c.Logger.Info("Timezone: " + fmt.Sprint(time.Local))
			c.Logger.Info("SHA: https://github.com/robbymilo/rgallery/commit/" + Commit)
			c.Logger.Info("Version: " + Tag)
			c.Logger.Info("rgallery listening on: " + port)
			err = http.ListenAndServe(":"+port, r)
			if err != nil {
				c.Logger.Error("error starting rgallery", "error", err)
			}

			return nil
		},
		Commands: []*cli.Command{
			{
				Name: "video-check", Usage: "Probe video acceleration and print diagnostics without scanning or starting the server.", Flags: flags,
				Before: validateVideoFlags,
				Action: func(cCtx *cli.Context) error {
					c := config.GetConf(*cCtx, Commit, Tag)
					c.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
					manager := transcode.NewManager(c)
					defer manager.Close()
					encoder := json.NewEncoder(os.Stdout)
					encoder.SetIndent("", "  ")
					return encoder.Encode(manager.Diagnostics())
				},
			},
			{
				Name:   "scan",
				Usage:  "Scan the media directory for new, modified, or delete media items.",
				Flags:  flags,
				Before: validateVideoFlags,
				Action: func(cCtx *cli.Context) error {
					c := config.GetConf(*cCtx, Commit, Tag)
					database.CreateDB(c)

					scanner.SetScanInProgress(false)

					cache := cache.New(-1, -1)
					_, err = scanner.Scan("default", c, cache)
					if err != nil {
						c.Logger.Error("error scanning", "error", err)
						os.Exit(1)
						return nil
					}

					return nil
				},
			},
			{
				Name:  "users",
				Usage: "Options for user tasks",
				Flags: flags,
				Subcommands: []*cli.Command{
					{
						Name:  "reset",
						Usage: "Remove all users and create an account with username 'admin' and password 'admin'.",
						Flags: flags,
						Action: func(cCtx *cli.Context) error {
							c := config.GetConf(*cCtx, Commit, Tag)

							err := users.ResetUsers(c)
							if err != nil {
								c.Logger.Error("error reseting users", "error", err)
								os.Exit(1)
								return nil
							}

							c.Logger.Info("users reset")
							return nil
						},
					},
					{
						Name:  "list",
						Usage: "Print a list of all users.",
						Flags: flags,
						Action: func(cCtx *cli.Context) error {
							c := config.GetConf(*cCtx, Commit, Tag)

							users, err := users.ListUsers(c)
							if err != nil {
								c.Logger.Error("error listing users", "error", err)
								os.Exit(1)
								return nil
							}

							fmt.Println("username", "role")
							for _, v := range users {
								fmt.Println(v.Username, v.Role)
							}

							return nil
						},
					},
					{
						Name:  "rm",
						Usage: "Remove a user.",
						Flags: flags,
						Action: func(cCtx *cli.Context) error {
							c := config.GetConf(*cCtx, Commit, Tag)

							creds := &UserCredentials{
								Username: cCtx.Args().Get(0),
							}

							err = users.RemoveUserConnect(creds, c)
							if err != nil {
								c.Logger.Error("error removing user", "error", err)
								os.Exit(1)
								return nil
							}

							c.Logger.Info("removed user " + cCtx.Args().Get(0))

							return nil
						},
					},
					{
						Name:  "add",
						Usage: "Add a user.",
						Flags: flags,
						Action: func(cCtx *cli.Context) error {
							c := config.GetConf(*cCtx, Commit, Tag)

							creds := &UserCredentials{
								Username: cCtx.Args().Get(0),
								Password: cCtx.Args().Get(1),
								Role:     cCtx.Args().Get(2),
							}

							err := users.AddUser(*creds, c)
							if err != nil {
								c.Logger.Error("error adding user", "error", err)
								os.Exit(1)
								return nil
							}

							c.Logger.Info("added user " + creds.Username)

							return nil
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
