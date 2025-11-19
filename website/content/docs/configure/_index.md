---
title: Configure rgallery
LinkTitle: Configure
weight: 300
---

# Configure rgallery

## Flags and environment variables

```shell
NAME:
   rgallery - A photo and video application.

USAGE:
   rgallery [global options] command [command options]

DESCRIPTION:
   The timeline for your photo and video library.

COMMANDS:
   scan     Scan the media directory for new, modified, or delete media items.
   users    Options for user tasks
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --dev                         Load assets from directory instead of embedding, allowing you to edit assets without a recompile. Disables caching of HTML and JSON responses. (default: false)
   --disable-auth                Load rgallery without a login. (default: false)
   --media value                 Location of the media directory. (default: "./media")
   --data value                  Location of the database directory (default: "./data")
   --cache value                 Location of the cache directory for storing image thumbnails and video transcode files. (default: "./cache")
   --config value                Location of the config yaml file. Only needed if using lens aliases. (default: "./config/config.yml")
   --quality value               Thumbnail resize quality. (default: 60)
   --transcode-resolution value  Resolution of transcoded videos. Defaults to 720p. For 1080p, set to 1920, for 4k set to 3840, for 8k set to 7680. Higher resolutions use more CPU and disk space. (default: 1280)
   --pregenerate-thumbs          Generate thumbnails and video transcode files during scan. Caution - may cause high server load if set to false. (default: true)
   --resize_service value        URL for resize service. [$RGALLERY_RESIZE_SERVICE]
   --location-service value      URL for reverse geocode service. [$RGALLERY_LOCATION_SERVICE]
   --location-dataset value      Dataset for reverse geocode lookup. Ex: Countries10, Countries110, Provinces10. Countries10 uses the least amount of memory, and Provinces10 the most. (default: "Provinces10")
   --tile-server value           URL for GeoServer tiles in XYZ format, ex https://tile.thunderforest.com/cycle/{z}/{x}/{y}.png?apikey=your-api-key-here. (default: "/tiles/{z}/{x}/{y}.png") [$RGALLERY_TILE_SERVER]
   --session-length value        Length of authenticated sessions in days. (default: 30) [$RGALLERY_SESSION_LENGTH]
   --include-originals           Include original files in web view. Setting this to true may cause slower image loading performance. (default: false)
   --on-this-day                 Show media items that occurred on the current day in previous years. (default: true)
   --help, -h                    show help
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
