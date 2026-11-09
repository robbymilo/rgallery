---
title: Scanning and importing media with rgallery
LinkTitle: Scanning
weight: 200
---

# Scanning and importing media

When rgallery is run for the first time, a default scan is started. During a default scan, the media directory is walked recursively, and media files are read and persisted, and thumbnails and video transcode files are generated.

## Supported extensions

rgallery will attempt to import files with the following extensions:

Images:

- .jpg
- .jpeg
- .heic
- .gif
- .png

Videos:

- .mp4
- .mov

When a subsequent scan is started, rgallery removes references to any files in the database that are no longer on disk. If any files have been updated they are reimported and thumbnails are regenerated.

There are three types of scans:

1. Default scan - regenerates thumbnails of modified media only.
1. Metadata scan - re-imports all existing media items without recreating thumbnails.
1. Deep scan - re-imports all existing media items and recreates thumbnails.

Only users with the role of **admin** can initiate scans from the web interface or manage scan errors. API keys have admin access and can be used for scheduled scans.

## Scan Errors

If an image or video fails during a scan, rgallery records the filepath and error in `scan_errors`. The item is skipped on later scans while its modification time is not newer than the recorded error. Updating the file causes it to be retried automatically.

An administrator can also clear the retry marker explicitly:

```shell
# Reset one path. The path is relative to the media directory.
rgallery scan-errors reset --path 'photos/failed-photo.jpg'

# The filepath may also be provided as the positional argument.
rgallery scan-errors reset 'photos/failed-photo.jpg'

# Allow every recorded scan error to be retried.
rgallery scan-errors reset --all
```

The same operations are available in the web UI under **Admin → Scan errors**. The page lists errors in pages, supports resetting one item, and has a reset-all action.
