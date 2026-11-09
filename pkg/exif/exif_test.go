package exif

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	exiftool "github.com/barasher/go-exiftool"
	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringToDate(t *testing.T) {
	time_strings := make(map[string]string)
	time_strings["1404139521877"] = "2014-06-30 14:45:21.877Z"
	time_strings["1404139521877.jpg"] = "2014-06-30 14:45:21.877Z"
	time_strings["IMG_20230924_142148.jpg"] = "2023-09-24 14:21:48Z"
	time_strings["IMG_19990924_142148.jpg"] = "1999-09-24 14:21:48Z"
	time_strings["IMG_20201016_220920_128.jpg"] = "2020-10-16 22:09:20Z"
	time_strings["2014-06-11_14.26.04.jpg"] = "2014-06-11 14:26:04Z"
	time_strings["20190323-wedding-03.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323-wedding.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323-1.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323-2.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323_wedding-03.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323-wedding_03.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323_wedding_03.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323_wedding.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323_1.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323_2.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323.jpg"] = "2019-03-23 00:00:00Z"

	for filename, ts := range time_strings {
		expected, _ := time.Parse("2006-01-02 15:04:05.999Z", ts)
		actual, _ := stringToDate(filename)
		assert.EqualValues(t, expected, actual, "they should be equal")
	}

}

func TestGetImageExifPreservesRawFieldsAndConvertedLens(t *testing.T) {
	relativePath := "2016/boise/20160424-boise-robbymilo-0123.jpg"
	absolutePath := filepath.Join("..", "..", "testdata", "media", relativePath)
	buffer := make([]byte, 1024*1024)
	raw, err := exiftool.NewExiftool(exiftool.NoPrintConversion(), exiftool.Buffer(buffer, 256*1024))
	require.NoError(t, err)
	defer func() { require.NoError(t, raw.Close()) }()
	converted, err := exiftool.NewExiftool(exiftool.Buffer(buffer, 256*1024))
	require.NoError(t, err)
	defer func() { require.NoError(t, converted.Close()) }()

	media, _, err := GetImageExif(
		"image",
		relativePath,
		absolutePath,
		raw,
		converted,
		nil,
		types.Conf{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))},
		nil,
	)
	require.NoError(t, err)

	assert.Equal(t, "AF Nikkor 50mm f/1.4D", media.Lens)
	assert.Equal(t, "1/1000", media.ShutterSpeed)
	assert.Equal(t, 2.0, media.Aperture)
	assert.Equal(t, 200.0, media.Iso)
	assert.Equal(t, 50.0, media.Focallength)
	assert.Equal(t, 50.0, media.FocalLength35)
	assert.Equal(t, time.Date(2016, time.April, 25, 2, 52, 27, 730000000, time.UTC), media.Date)
}
