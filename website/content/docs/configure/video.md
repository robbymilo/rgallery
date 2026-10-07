---
title: Video playback
---

# Video playback

rgallery prepares videos for fast playback in a web browser. By default, it creates a smaller chunks video when someone presses Play, then saves the
result so future playback is faster.

Video settings do not affect image thumbnail quality.

## Choose a quality

| Quality | Best for                                 | Maximum size     |
| ------- | ---------------------------------------- | ---------------- |
| `saver` | Slow or limited connections              | 854 px           |
| `small` | Most libraries (default)                 | 1280 px          |
| `high`  | Large screens and high-quality originals | Configured limit |

The size is the longest edge of the video. For example, a 1280 px limit usually produces 1280×720 video. rgallery does not enlarge smaller videos and preserves portrait orientation.

Set the default quality and maximum size when starting rgallery:

```sh
rgallery --transcode-quality=small --transcode-resolution=1920
```

A 1920 px limit allows Full HD output when **High** is selected. rgallery persists quality choice and playback position to the browser session.

## Choose when videos are prepared

Use `--transcode-mode` to control when rgallery prepares videos:

- `ondemand` (default): prepare a video when played.
- `hybrid`: prepare the default quality during a library scan and prepare other qualities when requested.
- `pregenerate`: prepare every quality during a library scan.

On-demand mode uses less storage and makes scans faster. Pregeneration makes first playback faster but increases scan time and storage use.

Seek thumbnails and short hover previews are prepared during scans when`--pregenerate-thumbs` is enabled. Missing previews can also be created when
requested.

## Storage and performance

Cached video files are stored under `cache/video/`. rgallery removes cached versions when the source file or encoding settings change. Original media files are never changed.

Adjust the number of simultaneous video jobs if needed:

```sh
rgallery --transcode-workers=2
```

More workers can prepare videos faster, but use more CPU or GPU resources.

## Hardware acceleration

rgallery can use Intel or AMD graphics hardware through VA-API on Linux. The default `auto` setting looks for a usable graphics device and falls back to CPU
encoding if none is available.

Check which encoder rgallery will use:

```sh
rgallery video-check
```

- `h264_vaapi` means hardware acceleration is active.
- `libx264` means CPU encoding is active.

To require a specific device:

```sh
rgallery video-check --transcode-encoder=vaapi \
  --transcode-device=/dev/dri/renderD128
```

The Admin → Video playback page also shows the active encoder, current jobs, encoding speed, and cache use.

## Common settings

| Setting                  | Default    | Purpose                                |
| ------------------------ | ---------- | -------------------------------------- |
| `--transcode-quality`    | `small`    | Default playback quality               |
| `--transcode-resolution` | `1280`     | Maximum long edge in pixels            |
| `--transcode-mode`       | `ondemand` | When videos are prepared               |
| `--transcode-encoder`    | `auto`     | Use automatic, CPU, or VA-API encoding |
| `--transcode-device`     | automatic  | Select a VA-API render device          |
| `--transcode-workers`    | `2`        | Maximum simultaneous video jobs        |
