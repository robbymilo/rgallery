---
title: rgallery FAQs
LinkTitle: FAQs
weight: 600
---

# rgallery FAQs

## Why does rgallery not scan raw photos?

The purpose of raw photos is to edit and process files in a raw photo editor before being used in their final display. Raw photos are often quite different from what one would expect from a photo, and need editing to look decent.

## How do I mark an image or video as a favorite?

Using exiftool, run the following command in your terminal:

```shell
exiftool -Rating=5 <path-to-media-file> -overwrite_original
```

Then initiate a scan in rgallery. The media item will appear in favorites.

## How do I add a title, description, and tags to an image or video?

Here is an example command to add multiple tags, a title, and a description to a file:

```shell
exiftool \
  -Subject="Nature" \
  -Subject="Sunset" \
  -Subject="Mountains" \
  -Title="Here is a title" \
  -Description="Here is a description" \
  <path-to-media-file> \
  -overwrite_original
```

It is also possible to import them into [darktable](https://www.darktable.org/), update the metadata and export final images.
