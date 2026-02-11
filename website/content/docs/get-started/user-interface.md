---
title: rgallery user interface
LinkTitle: User interface
---

# User interface

## Timeline page

The Timeline displays your media library with items grouped by day. You can search in the filter bar and filter by star rating. By default, items with 0 or 1 star and above are shown. You can refine the filter to show items rated 2, 3, 4, or 5 stars. You can also toggle between images and videos; the default view shows both.

Search supports EXIF fields, including reverse-geocoded location data. It also supports structured search syntax, such as:

- `tag:alpine-lakes-wilderness`
- `folder:2010/20100330-santa-cruz`
- `lens:AF Nikkor 85mm f/1.4D IF`
- `camera:NIKON Z 9`

On the right side, a scrubber lets you quickly navigate through your library. Drag it along the calendar to scroll to any point in your timeline. Each bar represents one month, and the length of the bar reflects how many media items are in that month.

{{< figure src="/rgallery-timeline-3.png" alt="Timeline page." >}}

## Memories page and panel.

If media items exist from previous years on the same day, they appear in a panel on the left side of the Timeline. This panel opens on hover or click. You can dismiss the memories for that day, and your choice is saved in the browser’s local storage.

Click “X days ago” or "View more" to scroll to that date in the Timeline.

{{< figure src="/ui/rgallery-memories.png" alt="Memories page." >}}

{{< figure src="/rgallery-memories.png" alt="Memories spine on the timeline page." >}}

## Favorites view

Click the fifth star in the filter bar to enable the Favorites view. This view shows all media items with an EXIF rating of 5.

{{< figure src="/ui/rgallery-favorites.png" alt="Favorites page." >}}

## Folders page

The Folders page displays a navigable list of all folders in the media directory.

{{< figure src="/ui/rgallery-folders.png" alt="Folders page." >}}

## Tags page

The Tags page displays a navigable list of all EXIF tags. It uses the EXIF “subject” field.

{{< figure src="/ui/rgallery-tags.png" alt="Tags page." >}}

## Gear page

The Gear page displays a navigable list of cameras, lenses, focal lengths, and more, along with the total number of media items.

{{< figure src="/ui/rgallery-gear.png" alt="Gear page." >}}

## Map page

The Map page displays all media items that include GPS coordinates on a map.

{{< figure src="/ui/rgallery-map.png" alt="Map page." >}}
