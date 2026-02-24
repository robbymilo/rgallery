---
title: Map tiles
---

# Map tiles

The map view shows geo tagged images on the world map. You can zoom and pan the map, and navigate to media items by clicking on their icons.

{{< figure src="/rgallery-map.png" alt="rgallery map" >}}

## Tile server

By default, a low-resolution, offline-only map tile server is used.

Use the `tile-server` CLI flag or `RGALLERY_TILE_SERVER` environmental variable to set a custom tile server URL.

To use a higher resolution map such as OpenStreetMap, tiles can be served from Thunderforest.

To use Thunderforest, create an account and obtain an API key.

You can then set the `tile-server` CLI flag or `RGALLERY_TILE_SERVER` environmental variable to the URL of the Thunderforest tile server.

For example:

```bash
RGALLERY_TILE_SERVER=https://tile.thunderforest.com/cycle/{z}/{x}/{y}.png?apikey=<replace-with-your-api-key>
```
