package server

import (
	"net/http"
)

// CheckResizeServiceHealth confirms the resize service is available to resize images.
func CheckResizeServiceHealth(c Conf) error {

	res, err := http.Get(c.ResizeService + "/healthz")
	if err != nil {
		return err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			c.Logger.Error("res.Body.Close error:", "err", err)
		}
	}()

	return nil
}

// CheckLocationServiceHealth confirms the resize service is available to resize images.
func CheckLocationServiceHealth(c Conf) error {

	res, err := http.Get(c.LocationService + "/healthz")
	if err != nil {
		return err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			c.Logger.Error("res.Body.Close error:", "err", err)
		}
	}()

	return nil
}
