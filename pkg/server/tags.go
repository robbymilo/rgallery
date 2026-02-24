package server

import (
	"fmt"
	"net/http"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/render"
	"github.com/robbymilo/rgallery/pkg/types"
)

type Subject = types.Subject
type Subjects = types.Subjects
type ResponseTags = types.ResponseTags
type ResponseMedia = types.ResponseMedia

func ServeTags(w http.ResponseWriter, r *http.Request) {

	params := r.Context().Value(ParamsKey{}).(FilterParams)
	c := r.Context().Value(ConfigKey{}).(Conf)

	cacheContext := r.Context().Value(CacheKey{}).(map[string]interface{})
	cacheHandle := cacheContext["cache"].(*cache.Cache)
	cacheMap, ok := cacheContext["response"].(*ResponseTags)
	if ok {
		err := render.Render(w, r, cacheMap, "index")
		if err != nil {
			c.Logger.Error("error rendering cached tags response", "error", err)
		}
		return
	}

	tags, err := queries.GetTags("subject", "asc", c)
	if err != nil {
		c.Logger.Error("error getting tags", "error", err)
	}
	total, err := queries.GetTotalTags(c)
	if err != nil {
		c.Logger.Error("error getting total of tags", "error", err)
	}

	response := ResponseTags{
		Tags:      tags,
		OrderBy:   "date",
		Page:      params.Page,
		PageSize:  -1,
		Total:     total,
		Direction: "asc",
		Section:   "tags",
		Meta:      c.Meta,
	}

	err = render.Render(w, r, response, "tags")
	if err != nil {
		c.Logger.Error("error rendering tags response", "error", err)
	}

	// setting cache
	cacheHandle.Set(fmt.Sprint(r.URL)+time.Now().Format("2006-01-02"), response, cache.NoExpiration)

}
