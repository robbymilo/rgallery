package exif

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	time_strings["20190323_wedding_03.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323_wedding.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323_1.jpg"] = "2019-03-23 00:00:00Z"
	time_strings["20190323_2.jpg"] = "2019-03-23 00:00:00Z"

	for filename, ts := range time_strings {
		expected, _ := time.Parse("2006-01-02 15:04:05.999Z", ts)
		actual, _ := stringToDate(filename)
		assert.EqualValues(t, expected, actual, "they should be equal")
	}

}
