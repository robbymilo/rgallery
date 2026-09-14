---
title: Configure rgallery
LinkTitle: Configure
weight: 300
---

# Configure rgallery

See [Video playback](video/) for quality profiles, streaming behavior, and Docker/Kubernetes GPU setup.

## Flags and environment variables

```shell
NAME:
   rgallery - A photo and video application.

USAGE:
   rgallery [global options] command [command options]

DESCRIPTION:
   The timeline for your photo and video library.

COMMANDS:
   video-check  Probe video acceleration and print diagnostics without scanning or starting the server.
   scan         Scan the media directory for new, modified, or delete media items.
   users        Options for user tasks
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --dev                            Load assets from directory instead of embedding, allowing you to edit assets without a recompile. Disables caching of HTML and JSON responses. (default: false)
   --disable-auth                   Load rgallery without a login. (default: false)
   --media value                    Location of the media directory. (default: "./media")
   --data value                     Location of the database directory (default: "./data")
   --cache value                    Location of the cache directory for storing image thumbnails and video transcode files. (default: "./cache")
   --config value                   Location of the config yaml file. Only needed if using lens aliases. (default: "./config/config.yml")
   --quality value                  Thumbnail resize quality. (default: 60)
   --transcode-resolution value     Maximum video long edge in pixels, without upscaling. 1280 allows 720p and 1920 allows 1080p for 16:9 sources. Individual quality profiles may use smaller dimensions. (default: 1280) [$RGALLERY_TRANSCODE_RESOLUTION]
   --transcode-quality value        Default video quality: saver, small, or high. (default: "small") [$RGALLERY_TRANSCODE_QUALITY]
   --transcode-mode value           Video generation: ondemand, pregenerate (all profiles), or hybrid (default profile during scan). Independent of thumbnail generation. (default: "ondemand") [$RGALLERY_TRANSCODE_MODE]
   --transcode-encoder value        Video encoder: auto, cpu, or vaapi. Hardware failures fall back to CPU. (default: "auto") [$RGALLERY_TRANSCODE_ENCODER]
   --transcode-device value         VA-API render device, e.g. /dev/dri/renderD128. Empty discovers accessible devices. [$RGALLERY_TRANSCODE_DEVICE]
   --transcode-crf value            CPU CRF for the default profile (0–51; higher means smaller files). -1 uses the profile default. (default: -1) [$RGALLERY_TRANSCODE_CRF]
   --transcode-preset value         CPU encoder speed preset; slower presets require more processing time. (default: "veryfast") [$RGALLERY_TRANSCODE_PRESET]
   --transcode-maxrate value        Video bitrate ceiling in kbps across all profiles. 0 uses each profile's ceiling. (default: 0) [$RGALLERY_TRANSCODE_MAXRATE]
   --transcode-audio-bitrate value  Audio bitrate in kbps (32–320). 0 uses each profile's default. (default: 0) [$RGALLERY_TRANSCODE_AUDIO_BITRATE]
   --transcode-workers value        Maximum concurrent video jobs (1–16). (default: 2) [$RGALLERY_TRANSCODE_WORKERS]
   --transcode-cache-mb value       Video cache budget in MiB (minimum 64). Active output is protected during eviction. (default: 10240) [$RGALLERY_TRANSCODE_CACHE_MB]
   --pregenerate-thumbs             Generate image thumbnails, video posters, and short video previews during scan. Full playback encoding is controlled by transcode-mode. (default: true)
   --resize_service value           URL for resize service. [$RGALLERY_RESIZE_SERVICE]
   --location-service value         URL for reverse geocode service. [$RGALLERY_LOCATION_SERVICE]
   --location-dataset value         Dataset for reverse geocode lookup. Ex: Countries10, Countries110, Provinces10. Countries10 uses the least amount of memory, and Provinces10 the most. (default: "Provinces10")
   --tile-server value              URL for GeoServer tiles in XYZ format, ex https://tile.thunderforest.com/cycle/{z}/{x}/{y}.png?apikey=your-api-key-here. (default: "/api/tiles/{z}/{x}/{y}.png") [$RGALLERY_TILE_SERVER]
   --session-length value           Length of authenticated sessions in days. (default: 30) [$RGALLERY_SESSION_LENGTH]
   --include-originals              Include original files in web view. Setting this to true may cause slower image loading performance. (default: false)
   --memories                       Show media items that occurred on the current day in previous years. (default: true)
   --help, -h                       show help
```

## Configuration file

The configuration file is a YAML file that should be located at `./config/config.yml`.

To use a different location, use the `--config` flag to specify the file location.

### Configuration file example

> Note: Only lens aliases and custom HTML are currently supported in the configuration file. Global options must use command line flags or, in some cases, environment variables.

```yaml
aliases:
  lenses: # exif:alias
    '17-35mm f/2.8-4E': Tamron 17-35mm f/2.8-4 Di OSD
    'Tamron 17-35mm f/2.8-4 Di OSD (A037)': Tamron 17-35mm f/2.8-4 Di OSD
    'Tamron 17-35mm f/2.8-4 Di OSD': Tamron 17-35mm f/2.8-4 Di OSD
    'Nikon 105mm f/2.5 Ai-s': 'Nikon Ai-s 105mm f/2.5'
    'Nikon 105mm f/2.5 AI-s': 'Nikon Ai-s 105mm f/2.5'
    'Nikon AI-s 105mm f/2.5': 'Nikon Ai-s 105mm f/2.5'
    'Nikon AI-s 105mm f/2.5   ': 'Nikon Ai-s 105mm f/2.5'
    'Nikon 105mm f/2.5 Ai-s ': 'Nikon Ai-s 105mm f/2.5'
    'VR 70-200mm f/2.8G': 'AF-S Nikkor 70-200mm f/2.8G ED VR II'
    'AF-S Nikkor 70-200mm f/2.8G ED VR II': 'AF-S Nikkor 70-200mm f/2.8G ED VR II'
custom_html: | # added before the closing body tag
  <script>
    console.log('custom html');
  </script>
```
