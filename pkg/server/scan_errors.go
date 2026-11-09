package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/robbymilo/rgallery/pkg/scanner"
)

const (
	defaultScanErrorsPageSize = 25
	maxScanErrorsPageSize     = 100
)

type scanErrorsResponse struct {
	Items []scanner.ScanError `json:"items"`
	Meta  scanErrorsMeta      `json:"meta"`
}

type scanErrorsMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

func ServeScanErrors(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)
	page, pageSize, err := scanErrorsPagination(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := scanner.GetScanErrorsPage(c, page, pageSize)
	if err != nil {
		c.Logger.Error("error listing scan errors", "error", err)
		http.Error(w, "Error getting scan errors", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(scanErrorsResponse{
		Items: result.Items,
		Meta:  scanErrorsMeta{Page: page, PageSize: pageSize, Total: result.Total},
	})
}

func ResetScanErrors(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)
	path := r.URL.Query().Get("path")
	resetAll := r.URL.Query().Get("all") == "true"
	if (path == "") == !resetAll {
		http.Error(w, "provide exactly one of path or all=true", http.StatusBadRequest)
		return
	}
	var err error
	if resetAll {
		err = scanner.ResetAllScanErrors(c)
	} else {
		err = scanner.ResetScanError(path, c)
	}
	if err != nil {
		c.Logger.Error("error resetting scan errors", "error", err)
		http.Error(w, fmt.Sprintf("Error resetting scan errors: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func scanErrorsPagination(r *http.Request) (page, pageSize int, err error) {
	page, pageSize = 1, defaultScanErrorsPageSize
	if raw := r.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 {
			return 0, 0, fmt.Errorf("page must be a positive integer")
		}
	}
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil || pageSize < 1 || pageSize > maxScanErrorsPageSize {
			return 0, 0, fmt.Errorf("pageSize must be between 1 and %d", maxScanErrorsPageSize)
		}
	}
	return page, pageSize, nil
}
