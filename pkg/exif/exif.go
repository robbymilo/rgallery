package exif

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/araddon/dateparse"
	exiftool "github.com/barasher/go-exiftool"
	"github.com/cenkalti/dominantcolor"
	"github.com/robbymilo/rgallery/pkg/geo"
	"github.com/robbymilo/rgallery/pkg/hash"
	"github.com/robbymilo/rgallery/pkg/resize"
	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/sams96/rgeo"
)

type Media = types.Media
type Subject = types.Subject
type imaging = image.Image
type Conf = types.Conf

// GetLens starts a new exiftool instance to retrieve the lens name. A new instance is needed as we pass "NoPrintConversion" to the other exiftool instance, which returns lens names such as "B0 00 50 50 14 14 B2 06" or "50 50 1.8 1.8 6" instead of the desired "AF-S Nikkor 50mm f/1.8G".
func GetLens(absolute_path string) (string, error) {
	buf := make([]byte, 1024*1024)
	et, err := exiftool.NewExiftool(exiftool.Buffer(buf, 256*1024))
	if err != nil {
		return "", fmt.Errorf("error starting exiftool: %v", err)
	}
	defer et.Close()

	exif := et.ExtractMetadata(absolute_path)

	var lens string
	for _, fileInfo := range exif {

		if fileInfo.Fields["LensModel"] != nil {
			lens = fileInfo.Fields["LensModel"].(string)
		} else if fileInfo.Fields["LensID"] != nil {
			lens = fileInfo.Fields["LensID"].(string)
		}

	}

	return lens, nil
}

// GetImageExif returns the exif data of a media item.
func GetImageExif(mediatype, relative_path, absolute_path string, et *exiftool.Exiftool, h *geo.Handlers, c Conf) (Media, imaging, error) {
	// relative_path is everything after the media dir
	hash := hash.GetHash(relative_path)

	exif := et.ExtractMetadata(absolute_path)

	file, err := os.Stat(absolute_path)
	if err != nil {
		return Media{}, nil, fmt.Errorf("error opening file: %v", err)
	}

	var media Media
	var img imaging
	var width int
	var height int
	var color string
	var rotation float64

	if mediatype == "image" {
		// need to open separately from exiftool to get correct orientation
		img, err = resize.DecodeImage(absolute_path)
		if err != nil {
			return Media{}, nil, err
		}

		// dominant color
		color = dominantcolor.Hex(dominantcolor.Find(img))

		// orientation
		bounds := img.Bounds()
		width = bounds.Dx()
		height = bounds.Dy()

		for _, fileInfo := range exif {
			if fileInfo.Fields["Rotation"] != nil {
				rotation = fileInfo.Fields["Rotation"].(float64)
			}
		}

	} else if mediatype == "video" {
		for _, fileInfo := range exif {

			if fileInfo.Fields["Rotation"] != nil {
				rotation = fileInfo.Fields["Rotation"].(float64)
			}

			if fileInfo.Fields["ImageWidth"] != nil {
				w := fileInfo.Fields["ImageWidth"].(float64)
				width = int(w)
			}

			if fileInfo.Fields["ImageHeight"] != nil {
				h := fileInfo.Fields["ImageHeight"].(float64)
				height = int(h)
			}

			if rotation == 90 || rotation == 270 {
				w := width
				h := height

				width = h
				height = w
			}
		}

	}

	for _, fileInfo := range exif {

		if fileInfo.Err != nil {
			return Media{}, nil, fmt.Errorf("error getting fileinfo: %v", fileInfo.Err)
		}

		date_string := fmt.Sprint(fileInfo.Fields["SubSecDateTimeOriginal"])
		if date_string == "" || date_string == "<nil>" {
			date_string = fmt.Sprint(fileInfo.Fields["SubSecCreateDate"])

			if date_string == "" || date_string == "<nil>" {
				date_string = fmt.Sprint(fileInfo.Fields["DateTimeOriginal"])

				if date_string == "" || date_string == "<nil>" {
					date_string = fmt.Sprint(fileInfo.Fields["TrackCreateDate"])

					if date_string == "" || date_string == "<nil>" {
						d, err := stringToDate(filepath.Base(absolute_path))
						if err != nil {
							return Media{}, nil, fmt.Errorf("error parsing date from filename: %v", err)
						}
						date_string = d.Format("2006-01-02T15:04:05.000Z")

						if date_string == "" || date_string == "<nil>" {
							return Media{}, nil, fmt.Errorf("media item has no date field :(")
						}
					}
				}
			}
		}

		// get UTC offset
		var dateOriginal time.Time
		var offsetMinutes float64
		if fileInfo.Fields["TimeZone"] != nil {
			// if there is an exif timezone field, we assume that is most accurate
			offsetMinutes = fileInfo.Fields["TimeZone"].(float64)

			// if daylight savings is set, add an hour to the offset
			dayLightSavings := fileInfo.Fields["DaylightSavings"].(float64)
			if dayLightSavings == 1 {
				offsetMinutes = offsetMinutes + 60
			}
			dateOriginal, err = getTimeUTC(date_string, offsetMinutes)
			if err != nil {
				return Media{}, nil, err
			}
		} else if strings.Contains(date_string, "+") {
			// get offset from end of date string, ex +0:00
			offset := strings.Split(date_string, "+")[1]
			offsetMinutes, err = ParseOffsetString(offset)
			if err != nil {
				return Media{}, nil, err
			}
			dateOriginal, err = getTimeUTC(date_string, offsetMinutes)
			if err != nil {
				return Media{}, nil, err
			}

		} else if fileInfo.Fields["GPSDateTime"] != nil {
			// compare gps time with date stamp to get an offset
			gps := fileInfo.Fields["GPSDateTime"].(string)
			dateOriginal, err = dateparse.ParseStrict(gps)
			if err != nil {
				return Media{}, nil, err
			}

			date, err := dateparse.ParseStrict(date_string)
			if err != nil {
				return Media{}, nil, err
			}

			offsetMinutes = date.Sub(dateOriginal).Minutes()

		} else {
			dateOriginal, err = dateparse.ParseStrict(date_string)
			if err != nil {
				return Media{}, nil, err
			}

		}

		ratio := float32(height) / float32(width)
		padding := ratio * 100
		subject := make([]Subject, 0)

		var rating float64
		if fileInfo.Fields["Rating"] != nil {
			rating = fileInfo.Fields["Rating"].(float64)
		}

		if fileInfo.Fields["Subject"] != nil {
			switch reflect.TypeOf(fileInfo.Fields["Subject"]).Kind() {
			case reflect.String:
				v := fmt.Sprint(fileInfo.Fields["Subject"])

				k := strings.Replace(strings.ToLower(v), " ", "-", -1)
				var r Subject
				r.Key = k
				r.Value = v

				subject = append(subject, r)
			case reflect.Slice:
				s := reflect.ValueOf(fileInfo.Fields["Subject"])
				for i := 0; i < s.Len(); i++ {
					v := fmt.Sprint(s.Index(i))
					k := strings.Replace(strings.ToLower(v), " ", "-", -1)

					var r Subject
					r.Key = k
					r.Value = v

					subject = append(subject, r)

				}
			}
		}

		var shutterRaw float64
		var shutterSpeed string
		if fileInfo.Fields["ShutterSpeed"] != nil {
			shutterRaw = fileInfo.Fields["ShutterSpeed"].(float64)
			shutterSpeed = floatToFractionString(shutterRaw)
		}

		var aperture float64
		if fileInfo.Fields["Aperture"] != nil {
			aperture = fileInfo.Fields["Aperture"].(float64)
		}

		var iso float64
		if fileInfo.Fields["ISO"] != nil {
			switch v := fileInfo.Fields["ISO"].(type) {
			case float64:
				iso = v
			case string:
				v = numbersOnly.ReplaceAllString(v, "")
				iso, err = strconv.ParseFloat(v, 64)
				if err != nil {
					return Media{}, nil, fmt.Errorf("error parsing ISO value: %v", err)
				}
			}
		}

		lens, err := GetLens(absolute_path)
		if err != nil {
			return Media{}, nil, fmt.Errorf("error getting lens exif: %v", err)
		}

		var camera string
		if fileInfo.Fields["Model"] != nil {
			camera = fileInfo.Fields["Model"].(string)
			// camera = cases.Title(language.English, cases.Compact).String(camera)
		}

		var focallength float64
		if fileInfo.Fields["FocalLength"] != nil {
			focallength = fileInfo.Fields["FocalLength"].(float64)
		}

		var focallength35 float64
		if fileInfo.Fields["FocalLengthIn35mmFormat"] != nil {
			focallength35 = fileInfo.Fields["FocalLengthIn35mmFormat"].(float64)
		}

		var altitude float64
		if fileInfo.Fields["GPSAltitude"] != nil {
			switch fileInfo.Fields["GPSAltitude"].(type) {
			case float64:
				altitude = fileInfo.Fields["GPSAltitude"].(float64)
			}
		}

		var latitude float64
		if fileInfo.Fields["GPSLatitude"] != nil {
			switch fileInfo.Fields["GPSLatitude"].(type) {
			case float64:
				latitude = fileInfo.Fields["GPSLatitude"].(float64)
			}
		}

		var longitude float64
		if fileInfo.Fields["GPSLongitude"] != nil {
			switch fileInfo.Fields["GPSLongitude"].(type) {
			case float64:
				longitude = fileInfo.Fields["GPSLongitude"].(float64)
			}
		}

		var focusdistance float64
		if fileInfo.Fields["FocusDistance"] != nil {
			focusdistance = fileInfo.Fields["FocusDistance"].(float64)
		}

		location := ""
		if longitude != 0 && latitude != 0 {

			var loc rgeo.Location
			if c.LocationService == "" {
				loc, err = geo.GetLocation(h, longitude, latitude, c)
				if err != nil {
					return Media{}, nil, fmt.Errorf("error getting location: %v", err)

				}
			} else {
				loc, err = geo.ReverseGeocode(longitude, latitude, c)
				if err != nil {
					return Media{}, nil, fmt.Errorf("error getting location from location service: %v", err)
				}
			}

			city := loc.City
			if city == "" && loc.Province != "" {
				city = fmt.Sprintf("%s,", loc.Province)
			} else if city != "" && loc.Province != "" {
				city = fmt.Sprintf("%s, %s,", loc.City, loc.Province)
			}
			location = fmt.Sprintf("%s %s", city, loc.Country)
		}

		description := ""
		if fileInfo.Fields["Description"] != nil {
			description = fileInfo.Fields["Description"].(string)
		} else if fileInfo.Fields["ImageDescription"] != nil {
			description = fileInfo.Fields["ImageDescription"].(string)
		}

		title := ""
		if fileInfo.Fields["Title"] != nil {
			title = fileInfo.Fields["Title"].(string)
		}

		software := ""
		if fileInfo.Fields["Software"] != nil {
			if _, ok := fileInfo.Fields["Software"].(string); ok {
				software = fileInfo.Fields["Software"].(string)
			}
		}

		media = Media{
			Hash:          hash,
			Path:          relative_path,
			Subject:       subject,
			Width:         width,
			Height:        height,
			Ratio:         ratio,
			Padding:       padding,
			Date:          dateOriginal,
			Modified:      file.ModTime().UTC(),
			Folder:        filepath.Dir(relative_path),
			Rating:        rating,
			ShutterSpeed:  shutterSpeed,
			Aperture:      aperture,
			Iso:           iso,
			Lens:          lens,
			Camera:        camera,
			Focallength:   focallength,
			Altitude:      altitude,
			Latitude:      latitude,
			Longitude:     longitude,
			Type:          mediatype,
			FocusDistance: focusdistance,
			FocalLength35: focallength35,
			Color:         color,
			Location:      location,
			Description:   description,
			Title:         title,
			Software:      software,
			Offset:        offsetMinutes,
			Rotation:      rotation,
		}
	}

	return media, img, err

}

var numbersOnly = regexp.MustCompile(`[^0-9]+`)

// stringToDate attempts to get a time from an arbitrary string such as a filename.
func stringToDate(date_string string) (time.Time, error) {

	// try for unix time (only when the numeric token is long enough to be milliseconds)
	unix_string := strings.Split(date_string, ".")
	if len(unix_string[0]) >= 11 {
		t, err := unixStringToTime(unix_string[0])
		if err == nil {
			return t, nil
		}
	}

	// try for YMD variants
	var raw_sub string

	if strings.Contains(date_string, "20") {
		raw_sub = date_string[strings.Index(date_string, "20"):]
	} else if strings.Contains(date_string, "19") {
		raw_sub = date_string[strings.Index(date_string, "19"):]
	} else {
		raw_sub = date_string
	}

	// Collect digits and common separators until we hit a letter,
	// so we don't accidentally append unrelated trailing numbers.
	var buf strings.Builder
	for _, r := range raw_sub {
		if unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			buf.WriteRune(r)
		} else if unicode.IsLetter(r) {
			// stop collecting once we hit letters (e.g. "-wedding..."), keep what we have so far
			break
		} else {
			// other characters: skip but continue collecting to allow patterns like "2014-06-11_14.26.04"
			continue
		}
	}

	time_string := buf.String()
	// remove separators
	time_string = strings.ReplaceAll(time_string, "_", "")
	time_string = strings.ReplaceAll(time_string, "-", "")
	time_string = strings.ReplaceAll(time_string, ".", "")
	time_string = numbersOnly.ReplaceAllString(time_string, "")

	// Need at least YYYYMMDD; try parsing raw_sub if too short
	if len(time_string) < 8 {
		tp, err := dateparse.ParseAny(raw_sub)
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(tp.Year(), tp.Month(), tp.Day(), tp.Hour(), tp.Minute(), tp.Second(), tp.Nanosecond(), time.UTC), nil
	}

	// Truncate to a maximum of 14 characters (YYYYMMDDHHMMSS)
	if len(time_string) > 14 {
		time_string = time_string[:14]
	}

	// If the numeric string length isn't one of the expected lengths,
	// prefer the YYYYMMDD date-only form by truncating to 8 digits.
	if !(len(time_string) == 8 || len(time_string) == 10 || len(time_string) == 12 || len(time_string) == 14) {
		if len(time_string) > 8 {
			time_string = time_string[:8]
		}
	}

	var layout string
	switch len(time_string) {
	case 14:
		layout = "20060102150405"
	case 12:
		layout = "200601021504"
	case 10:
		layout = "2006010215"
	case 8:
		layout = "20060102"
	default:
		// fallback to dateparse if unusual length
		tp, err := dateparse.ParseAny(time_string)
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(tp.Year(), tp.Month(), tp.Day(), tp.Hour(), tp.Minute(), tp.Second(), tp.Nanosecond(), time.UTC), nil
	}

	// Parse with chosen layout
	parsed, err := time.Parse(layout, time_string)
	if err != nil {
		// fallback to dateparse
		tp, err2 := dateparse.ParseAny(time_string)
		if err2 != nil {
			return time.Time{}, err
		}
		return time.Date(tp.Year(), tp.Month(), tp.Day(), tp.Hour(), tp.Minute(), tp.Second(), tp.Nanosecond(), time.UTC), nil
	}

	// Return the same wall-clock time but set to UTC (avoid applying local TZ offset)
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(), time.UTC), nil
}

func unixStringToTime(u string) (time.Time, error) {
	i, err := strconv.ParseInt(u, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	time := time.UnixMilli(i).UTC()

	return time, nil
}

// getTimeUTC takes a date string and offset minutes and returns a time in UTC.
func getTimeUTC(dateString string, offset float64) (time.Time, error) {
	// Parse the dateString to time
	t, err := dateparse.ParseStrict(dateString)
	if err != nil {
		return time.Time{}, err
	}

	tz := time.FixedZone("", (int(offset) * 60))
	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)

	utc := t.UTC()
	return utc, nil
}

// ParseOffsetString takes an offset string such as "+2:00" and returns a duration in minutes.
func ParseOffsetString(t string) (float64, error) {
	var mins, hours int
	var err error

	parts := strings.SplitN(t, ":", 2)

	switch len(parts) {
	case 1:
		mins, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
	case 2:
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}

		mins, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("invalid time: %s", t)
	}

	if mins > 59 || mins < 0 || hours > 23 || hours < 0 {
		return 0, fmt.Errorf("invalid time: %s", t)
	}

	return (time.Duration(hours)*time.Hour + time.Duration(mins)*time.Minute).Minutes(), nil
}

// floatToFractionString converts a float to a fraction string.
func floatToFractionString(f float64) string {
	// Precision for float comparison
	epsilon := 1e-9

	// Multiply by a large number to handle precision loss
	num := int(f * 1000000)
	denom := 1000000

	// Reduce the fraction
	divisor := gcd(num, denom)
	num /= divisor
	denom /= divisor

	// Check if the float can be represented exactly as a fraction
	if math.Abs(float64(num)/float64(denom)-f) < epsilon {
		return fmt.Sprintf("%d/%d", num, denom)
	}

	// If the float cannot be represented exactly, approximate it
	// Find the closest fraction to the input float
	num, denom = 1, 1
	bestError := math.Abs(f - float64(num)/float64(denom))
	for i := 1; i <= 1000; i++ {
		for j := 1; j <= 1000; j++ {
			if math.Abs(f-float64(i)/float64(j)) < bestError {
				num, denom = i, j
				bestError = math.Abs(f - float64(i)/float64(j))
			}
		}
	}

	return fmt.Sprintf("%d/%d", num, denom)

}

// Function to find the greatest common divisor (GCD) of two integers
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
