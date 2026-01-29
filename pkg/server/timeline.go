package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/render"
	"github.com/robbymilo/rgallery/pkg/types"
)

type Media = types.Media
type Days = types.Days
type Filter = types.Filter
type ResponseFilter = types.ResponseFilter

type CacheKey = types.CacheKey

func ServeTimeline(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)

	params := r.Context().Value(ParamsKey{}).(FilterParams)
	var response *queries.TimelineResponse
	w.Header().Set("Access-Control-Allow-Origin", "*")

	cacheContext := r.Context().Value(CacheKey{}).(map[string]interface{})
	cacheHandle := cacheContext["cache"].(*cache.Cache)
	cacheMap, ok := cacheContext["response"].(*queries.TimelineResponse)
	if ok {
		err := render.RenderJson(w, r, cacheMap)
		if err != nil {
			c.Logger.Error("error rendering cached timeline response", "error", err)
		}
		return
	}

	cursorQuery := r.URL.Query().Get("cursor")
	if cursorQuery != "" {
		var err error
		params.Cursor, err = strconv.Atoi(cursorQuery)
		if err != nil {
			params.Cursor = 0
		}
	} else {
		params.Cursor = 0
	}

	params.PageSize = 1000

	response, err := queries.GetTimeline(&params, c)
	if err != nil {
		c.Logger.Error("error getting timeline", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = render.RenderJson(w, r, response)
	if err != nil {
		c.Logger.Error("error rendering timeline response", "error", err)
	}

	cacheHandle.Set(fmt.Sprint(r.URL)+time.Now().Format("2006-01-02"), response, cache.NoExpiration)
}
