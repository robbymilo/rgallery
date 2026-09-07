package server

import (
	"fmt"
	"net/http"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/render"
)

func ServeMemories(w http.ResponseWriter, r *http.Request) {
	params := r.Context().Value(ParamsKey{}).(FilterParams)
	c := r.Context().Value(ConfigKey{}).(Conf)

	cacheContext := r.Context().Value(CacheKey{}).(map[string]interface{})
	cacheHandle := cacheContext["cache"].(*cache.Cache)
	cacheMap, ok := cacheContext["response"].(Memories)
	if ok {
		err := render.RenderJson(w, r, cacheMap)
		if err != nil {
			c.Logger.Error("error rendering cached memories response", "error", err)
		}
		return
	}

	day := time.Now().Format("2006-01-02")
	if params.Date != "" {
		// verify YYYY-MM-DD format
		parsedDate, err := time.Parse("2006-01-02", params.Date)
		if err != nil {
			c.Logger.Error("error parsing date, using today instead", "error", err)
		}
		day = parsedDate.Format("2006-01-02")
	}

	memories := Memories{}
	var err error

	if c.Memories {
		memories, err = queries.GetMemories(day, c)
		if err != nil {
			c.Logger.Error("error getting memories", "error", err)
		}
	}

	err = render.RenderJson(w, r, memories)
	if err != nil {
		c.Logger.Error("error rendering memories", "error", err)
	}

	cacheKey := fmt.Sprint(r.URL) + day
	cacheHandle.Set(cacheKey, memories, cache.NoExpiration)

}
