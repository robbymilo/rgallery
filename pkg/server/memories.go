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
	c := r.Context().Value(ConfigKey{}).(Conf)

	cacheContext := r.Context().Value(CacheKey{}).(map[string]interface{})
	cacheHandle := cacheContext["cache"].(*cache.Cache)
	cacheMap, ok := cacheContext["response"].(Days)
	if ok {
		err := render.RenderJson(w, r, cacheMap)
		if err != nil {
			c.Logger.Error("error rendering cached memories response", "error", err)
		}
		return
	}

	days := Days{}
	var err error

	if c.Memories {
		days, err = queries.GetMemories(c)
		if err != nil {
			c.Logger.Error("error getting memories", "error", err)
		}
	}

	err = render.RenderJson(w, r, days)
	if err != nil {
		c.Logger.Error("error rendering memories", "error", err)
	}

	cacheHandle.Set(fmt.Sprint(r.URL)+time.Now().Format("2006-01-02"), days, cache.NoExpiration)

}
