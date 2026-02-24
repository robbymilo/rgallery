package sizes

import (
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"slices"
	"strings"

	"github.com/robbymilo/rgallery/pkg/types"
)

type Conf = types.Conf

func Srcset(hash uint32, width int, path string, c Conf) template.Srcset {
	var srcset string
	final := false

	for _, size := range GetSizes() {
		suffix := ", "
		if size <= width {
			srcset = fmt.Sprintf(`%s/api/img/%d/%d %dw%s`, srcset, hash, size, size, suffix)
		} else if !final {
			final = true
			srcset = fmt.Sprintf(`%s/api/img/%d/%d %dw%s`, srcset, hash, width, width, suffix)
		}
	}

	if c.IncludeOriginals {
		url := template.HTMLEscapeString(filepath.Join("/api/media-originals", path))
		srcset = fmt.Sprintf(`%s%s %dw`, srcset, url, width)
	}

	return template.Srcset(strings.TrimSuffix(srcset, ", "))
}

func GetSizes() []int {
	return []int{200, 400, 800, 1200, 1800, 2400, 4000}
}

// ValidThumbSize prevents creating a thumb that is not a specified size or a size larger than the original image.
func ValidThumbSize(size int, longEdge int) (bool, error) {
	var err error
	var valid bool

	if (slices.Contains(GetSizes(), size) && size <= longEdge) || ((size == longEdge) && longEdge <= 4000) {
		valid = true
	} else {
		err = errors.New("thumb request out of band")

	}

	return valid, err
}
