# rgallery

<img width="1800" height="1012" alt="rgallery-readme" src="https://github.com/user-attachments/assets/0871bbd6-3a5c-455a-90e5-0f19edf01729" />

## About

rgallery unifies your photo and video collection into a single, elegant web interface, complete with a timeline view, memories, favorites, folders, EXIF metadata, map view, and more. To get started, see [Deploy with Docker](#docker) or [visit the docs](https://rgallery.app/docs/) for more information.

rgallery was originally designed for photographer's with highly organized photo collections, but adapts to all photo and video collections, even if they are not well organized.

Try the demo: https://demo.rgallery.app (User: **demo**, Password: **demo**)

## Features

### Explore your photo and video library

- Timeline view: Scroll to any date in your library, all in a single view grouped by day. Filter by folder, tag, camera, lens, and other metadata.
- Search everything: Search metadata across all your images and videos.
- Map view: Explore your geo-tagged memories across the world.
- Reverse geocoding: Tag media items with cities and countries from EXIF coordinates without external API calls.
- Infinite slider: Swipe through your media library until the very end, effortlessly.
- Permalinks: Every image and video gets a unique, persistent URL.
- Folder view: A recursive look into your library's folder structure.
- Metadata at a glance: View detailed EXIF metadata for all media items.
- Dark mode.
- Memories: Revisit relevant events that happened "on this day."
- Gear stats: Analyze your media collection by camera, lens, focal length, and more. Have multiple cameras that record different EXIF tags for the same lens? Use [lens aliases](#lens-aliases) to view them together.

## Get started

### Deploy with docker

In a terminal, configure the path to your media files and run the following command:

```shell
docker run \
  -v /path-to-your-media-files:/media:ro \
  -v ./data:/data \
  -v ./cache:/cache \
  -p 3000:3000 \
  robbymilo/rgallery:latest
```

The following volume is required for rgallery to access your media files:

- path to your media files: `/path-to-your-media-files:/media:ro`

The following volumes are required to persist the database and image thumbnail cache:

- path to the database directory: `./data:/data`
- path to the cache directory: `./cache:/cache`

If they are unchanged the data and cache directories will be created in the current directory where the command is run.

The application will be available at [http://localhost:3000](http://localhost:3000). The default username and password are both **admin**.

To learn more, visit:

- [Deploy with Docker or Docker Compose](https://rgallery.app/docs/install/docker/)
- [Deploy with Kubernetes](https://rgallery.app/docs/install/kubernetes/)
- [Deploy with Tanka](https://rgallery.app/docs/install/tanka/)
- [Deploy with Linux](https://rgallery.app/docs/install/linux/)

## Roadmap

See https://github.com/robbymilo/rgallery/issues/15.
