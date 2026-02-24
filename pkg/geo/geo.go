package geo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/sams96/rgeo"
)

type Location = rgeo.Location
type Conf = types.Conf

type Handlers struct {
	r *rgeo.Rgeo
}

func NewGeoHandler(c Conf) (*Handlers, error) {
	c.Logger.Info("starting new geo handler...")
	var r *rgeo.Rgeo
	var err error

	switch c.LocationDataset {
	case "Provinces10":
		r, err = rgeo.New(rgeo.Provinces10)
		if err != nil {
			return nil, err
		}
	case "Countries110":
		r, err = rgeo.New(rgeo.Countries110)
		if err != nil {
			return nil, err
		}
	case "Countries10":
		r, err = rgeo.New(rgeo.Countries10)
		if err != nil {
			return nil, err
		}
	case "Cities10":
		r, err = rgeo.New(rgeo.Cities10)
		if err != nil {
			return nil, err
		}
	}

	c.Logger.Info("geo handler ready")
	return &Handlers{
		r: r,
	}, nil
}

func GetLocation(h *Handlers, lon, lat float64, c Conf) (Location, error) {

	loc, err := h.r.ReverseGeocode([]float64{lon, lat})
	if err != nil {
		c.Logger.Error("error getting location data", "error", err)
	}

	return loc, nil
}

func ReverseGeocode(lon, lat float64, c Conf) (Location, error) {
	baseURL, err := url.Parse(fmt.Sprintf("%s/?lon=%v&lat=%v", c.LocationService, lon, lat))
	if err != nil {
		return Location{}, err
	}

	response, err := http.Get(baseURL.String())
	if err != nil {
		return Location{}, err
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			c.Logger.Error("response.Body.Close error", "err", err)
		}
	}()

	jsonData, err := io.ReadAll(response.Body)
	if err != nil {
		return Location{}, err
	}

	var location Location
	err = json.Unmarshal([]byte(jsonData), &location)
	if err != nil {
		return Location{}, err
	}

	return location, nil

}
